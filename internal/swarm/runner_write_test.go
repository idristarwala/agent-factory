package swarm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/queue"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/tools"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

type failingGraph struct {
	*graph.MemoryGraph
	err error
}

func (g failingGraph) Apply(*graph.Mutation) error { return g.err }

type titleGateway struct{}

func (titleGateway) Generate(models.Request) (*models.Response, error) {
	return &models.Response{Content: `{"title":"fixed"}`}, nil
}

type writeTrackingQueue struct {
	*queue.MemoryQueue
	failed, completed int
}

func (q *writeTrackingQueue) Fail(ctx context.Context, id, reason string) error {
	q.failed++
	return q.MemoryQueue.Fail(ctx, id, reason)
}

func (q *writeTrackingQueue) Complete(ctx context.Context, id string) error {
	q.completed++
	return q.MemoryQueue.Complete(ctx, id)
}

func TestFailedGraphWriteNeverCompletesUnit(t *testing.T) {
	writeErr := errors.New("graph unavailable")
	g := failingGraph{graph.NewMemoryGraph(), writeErr}
	g.AddNode(graph.Node{ID: "item", Props: map[string]any{"status": "needs_fix", "title": "needs review"}})
	s := agents.NewSmartSteward(unitofwork.NewMemoryStore())
	s.SetRuntime(titleGateway{}, g, nil)
	q := &writeTrackingQueue{MemoryQueue: queue.NewMemoryQueue()}
	r := NewContinuousRunner(s, agents.NewLLMJudge(&evidenceGateway{}, g), q, g, rules.NewEngine(), nil)
	if err := r.Run(context.Background(), "fix title"); !errors.Is(err, writeErr) {
		t.Fatalf("expected graph write error, got %v", err)
	}
	entries := r.Audit.All()
	if len(entries) != 3 || entries[0].Action != "proposed" || entries[1].Action != "approved" || entries[2].Action != "apply_failed" || entries[0].TraceID == "" || entries[0].TraceID != entries[2].TraceID {
		t.Fatal(entries)
	}
	if q.failed != 1 || q.completed != 0 {
		t.Fatalf("failed %d, completed %d", q.failed, q.completed)
	}
}

type evidenceGateway struct {
	requests []models.Request
}

func (g *evidenceGateway) Generate(req models.Request) (*models.Response, error) {
	g.requests = append(g.requests, req)
	if req.CallerRole == "judge" {
		return &models.Response{Content: "APPROVE: verified"}, nil
	}
	return titleGateway{}.Generate(req)
}

func TestRunnerSharesAuthorizedEvidence(t *testing.T) {
	payload := strings.Repeat("x", 300) + " fact at end"
	for _, authorized := range []bool{false, true} {
		g := graph.NewMemoryGraph()
		g.AddNode(graph.Node{ID: "record", Props: map[string]any{"status": "needs_fix", "title": "needs review"}})
		gateway := &evidenceGateway{}
		steward := agents.NewSmartSteward(unitofwork.NewMemoryStore())
		steward.SetRuntime(gateway, g, nil)
		q := &writeTrackingQueue{MemoryQueue: queue.NewMemoryQueue()}
		r := NewContinuousRunner(steward, agents.NewLLMJudge(gateway, g), q, g, rules.NewEngine(), nil)
		r.Workers = 1
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if !authorized {
				t.Error("request made without grant")
			}
			w.Write([]byte(payload))
		}))
		tool := tools.NewHTTPTool("source", "GET", srv.URL)
		tool.Client = srv.Client()
		r.SetTools(map[string]*tools.HTTPTool{"source": tool}, tools.ContextRecipe{Entity: "source"})
		names := []string{"propose"}
		if authorized {
			names = append(names, "source")
		}
		r.SetGrants("review", []tools.Grant{{Pack: "review", Tools: names, SideEffects: []string{"read-only"}}})
		err := r.Run(t.Context(), "fix title")
		srv.Close()
		if !authorized {
			if err == nil || len(gateway.requests) != 0 || q.failed != 1 {
				t.Fatalf("denied case: err=%v calls=%d failures=%d", err, len(gateway.requests), q.failed)
			}
			continue
		}
		if err != nil || len(gateway.requests) != 2 || q.completed != 1 {
			t.Fatalf("authorized case: err=%v calls=%d completed=%d", err, len(gateway.requests), q.completed)
		}
		for _, req := range gateway.requests {
			if req.UnitID == "" || req.UnitID != r.Audit.All()[0].TraceID {
				t.Fatal("uncorrelated model request")
			}
			if !strings.Contains(req.User, "entity (tool source): "+payload) {
				t.Errorf("%s missing full source evidence", req.CallerRole)
			}
		}
	}
}
