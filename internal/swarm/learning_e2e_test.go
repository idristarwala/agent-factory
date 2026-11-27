package swarm

import (
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/queue"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/learning"
	"github.com/idristarwala/agent-factory/internal/rules"
)

func TestReplayGateHoldsLearnedRules(t *testing.T) {
	re := rules.NewEngine()
	loop := learning.NewLearningLoop(re, agents.NewRegistry())
	loop.ReplayGate = true

	decisions := []agents.Decision{
		{Approved: false, Reason: "format failure"},
		{Approved: false, Reason: "format failure"},
	}
	gaps := loop.Review(decisions, nil)
	updates := loop.ProposeUpdates(gaps)
	applied, err := loop.Apply(updates)
	if err != nil {
		t.Fatal(err)
	}

	if len(applied) == 0 {
		t.Fatal("expected a rule to be proposed")
	}
	active := re.Active()
	if len(active) != 0 {
		t.Fatalf("replay gate should hold the rule; got %d active", len(active))
	}
}

func TestSameCauseFailuresMakeOneRule(t *testing.T) {
	re := rules.NewEngine()
	loop := learning.NewLearningLoop(re, agents.NewRegistry())

	failures := []string{
		"format failure",
		"format failure",
		"format failure",
	}
	gaps := loop.Review(nil, failures)
	updates := loop.ProposeUpdates(gaps)
	loop.Apply(updates)

	active := re.Active()
	if len(active) != 1 {
		t.Fatalf("expected one rule for the same cause; got %d", len(active))
	}
}

func TestRejectionsAcrossRoundsAreLearned(t *testing.T) {
	re := rules.NewEngine()
	loop := learning.NewLearningLoop(re, agents.NewRegistry())

	// Round 1: one rejection
	loop.Review([]agents.Decision{{Approved: false, Reason: "title quality"}}, nil)
	// Round 2: same theme
	gaps := loop.Review([]agents.Decision{{Approved: false, Reason: "title quality"}}, nil)
	if len(gaps) == 0 {
		t.Fatal("expected gaps after repeated rejections")
	}
	updates := loop.ProposeUpdates(gaps)
	loop.Apply(updates)

	active := re.ToAgentsRules()
	if len(active) == 0 {
		t.Fatal("expected an active rule for the next round")
	}
}

type learningGateway struct{ invalid, corrected, judged int }

func (g *learningGateway) Generate(req models.Request) (*models.Response, error) {
	if req.CallerRole == "judge" {
		g.judged++
		return &models.Response{Content: "APPROVE: matches source"}, nil
	}
	if strings.Contains(req.User, "Return only the allowed fields as flat JSON") {
		g.corrected++
		return &models.Response{Content: `{"assigned_team":"support"}`}, nil
	}
	g.invalid++
	return &models.Response{Content: `{"unrecognized":true}`}, nil
}

func TestScriptedRoundImprovement(t *testing.T) {
	g := graph.NewMemoryGraph()
	g.AddNode(graph.Node{ID: "request", Type: "ticket", Props: map[string]any{"status": "open", "assigned_team": "unknown"}})
	gateway := &learningGateway{}
	steward := agents.NewSmartSteward(unitofwork.NewMemoryStore())
	steward.SetRuntime(gateway, g, nil)
	re := rules.NewEngine()
	loop := learning.NewLearningLoop(re, agents.NewRegistry())
	q := &writeTrackingQueue{MemoryQueue: queue.NewMemoryQueue()}
	runner := NewContinuousRunner(steward, agents.NewLLMJudge(gateway, g), q, g, re, loop)
	runner.Workers = 1
	if err := runner.Run(t.Context(), "Assign the support team"); err != nil {
		t.Fatal(err)
	}
	record, err := g.GetNode("request")
	if err != nil || record.Props["assigned_team"] != "support" || record.Props["status"] != "fixed" {
		t.Fatal(record, err)
	}
	if q.failed != 1 || q.completed != 1 || gateway.invalid != 2 || gateway.corrected != 1 || gateway.judged != 1 || len(re.Active()) != 1 {
		t.Fatalf("failed=%d completed=%d model calls=%+v rules=%v", q.failed, q.completed, gateway, re.Active())
	}
}
