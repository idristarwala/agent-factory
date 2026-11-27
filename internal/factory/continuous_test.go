package factory

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/learning"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/obs"
	"github.com/idristarwala/agent-factory/internal/queue"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/settings"
	"github.com/idristarwala/agent-factory/internal/swarm"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func TestContinuousUsageConcurrentWorkersAndFailedRun(t *testing.T) {
	usage := &ContinuousUsage{Gateway: scripted{}, Prices: map[string]Price{"test": {Input: 1, Output: 2}}}
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			usage.Generate(models.Request{CallerRole: "worker", UnitID: "unit-1"})
			usage.RecordTool("source")
		})
	}
	workers.Wait()
	ws := Workspace{Store: revision.NewMemory()}
	audit := []obs.AuditEntry{{TraceID: "unit-1", Actor: "worker", Action: "proposed"}}
	run, err := ws.FinishContinuous("Review requests", time.Now(), settings.Settings{}, 3, usage, audit, errors.New("round limit"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := ws.Get(run.ID)
	if err != nil || saved.Error != "round limit" || saved.Status != "failed" || saved.SettingsRevision != 3 || len(saved.Calls) != 20 || usage.Tokens() != 240 || saved.Tools["source"] != 20 {
		t.Fatal(saved, err)
	}
	if len(saved.Audit) != 1 || saved.Audit[0].Action != "proposed" {
		t.Fatal(saved.Audit)
	}
	for _, call := range saved.Calls {
		if call.UnitID != saved.Audit[0].TraceID || call.Total != 12 || call.Cost != 0.000014 || call.Role != "worker" {
			t.Fatal(call)
		}
	}
}

type budgetGateway struct {
	requests chan models.Request
	replies  chan int
}

func (g *budgetGateway) Generate(req models.Request) (*models.Response, error) {
	g.requests <- req
	return &models.Response{Usage: map[string]int{"total_tokens": <-g.replies}}, nil
}
func TestContinuousBudgetReservations(t *testing.T) {
	gw := &budgetGateway{make(chan models.Request, 3), make(chan int)}
	usage := &ContinuousUsage{Gateway: gw, MaxTokens: 10}
	done := make(chan error, 3)
	call := func() { _, err := usage.Generate(models.Request{MaxTokens: 8}); done <- err }
	go call()
	if req := <-gw.requests; req.MaxTokens != 8 {
		t.Fatal(req)
	}
	go call()
	if req := <-gw.requests; req.MaxTokens != 2 {
		t.Fatal(req)
	}
	go call()
	gw.replies <- 10
	gw.replies <- 2
	var denied int
	for range 3 {
		if err := <-done; errors.Is(err, models.ErrTokenBudget) {
			denied++
		}
	}
	if denied != 1 || len(gw.requests) != 0 || len(usage.calls) != 2 || usage.Tokens() != 12 {
		t.Fatal(denied, usage.calls, usage.Tokens())
	}
}

func TestContinuousBudgetDenialDoesNotBecomeLearning(t *testing.T) {
	st := revision.NewMemory()
	re := rules.NewStoredEngine(st)
	usage := &ContinuousUsage{Gateway: scripted{}, MaxTokens: 12}
	g := graph.NewMemoryGraph()
	g.AddNode(graph.Node{ID: "ticket", Props: map[string]any{"status": "open", "priority": 1}})
	steward := agents.NewSmartSteward(unitofwork.NewMemoryStore())
	steward.AllowedFields = []string{"priority"}
	steward.SetRuntime(usage, g, nil)
	loop := learning.NewLearningLoop(re, agents.NewStoredRegistry(st))
	judge := agents.NewLLMJudge(usage, g)
	panel := agents.NewQuorumJudge([]agents.Judge{judge}, 1)
	runner := swarm.NewContinuousRunner(steward, panel, queue.NewMemoryQueue(), g, re, loop)
	runner.Workers = 1
	runner.Usage = usage
	err := runner.Run(t.Context(), "Set priority from evidence")
	if !errors.Is(err, models.ErrTokenBudget) || len(usage.calls) != 1 || usage.calls[0].Role != "worker" || len(re.History("")) != 0 {
		t.Fatal(err, usage.calls, re.History(""))
	}
	events := runner.Audit.All()
	if len(events) != 2 || events[1].Action != "budget_exhausted" {
		t.Fatal(events)
	}
	record, _ := g.GetNode("ticket")
	if record.Props["status"] != "open" || record.Props["priority"] != 1 {
		t.Fatal(record)
	}
}
