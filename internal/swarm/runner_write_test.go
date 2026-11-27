package swarm

import (
	"context"
	"errors"
	"testing"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/queue"
	"github.com/idristarwala/agent-factory/internal/rules"
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
	r := NewContinuousRunner(s, agents.NewStubJudge(), q, g, rules.NewEngine(), nil)
	if err := r.Run(context.Background(), "fix title"); !errors.Is(err, writeErr) {
		t.Fatalf("expected graph write error, got %v", err)
	}
	if q.failed != 1 || q.completed != 0 {
		t.Fatalf("failed %d, completed %d", q.failed, q.completed)
	}
}
