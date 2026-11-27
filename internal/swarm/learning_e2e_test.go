package swarm

import (
	"encoding/json"
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

type dispatchGateway struct{ focused int }

func (g *dispatchGateway) Generate(req models.Request) (*models.Response, error) {
	if req.CallerRole == "judge" {
		return &models.Response{Content: "APPROVE: grounded"}, nil
	}
	if strings.Contains(req.User, "Specialist focus:") && strings.Contains(req.User, "Worker instructions: Use routing evidence") {
		g.focused++
		return &models.Response{Content: `{"assigned_team":"support"}`}, nil
	}
	return &models.Response{Content: `{"unexpected":true}`}, nil
}

func TestSpecialistDispatchHonorsActiveRuleAndWorkerRevision(t *testing.T) {
	for _, gated := range []bool{false, true} {
		g := graph.NewMemoryGraph()
		for _, id := range []graph.NodeID{"request-a", "request-b"} {
			g.AddNode(graph.Node{ID: id, Type: "ticket", Props: map[string]any{"status": "open", "assigned_team": "unknown"}})
		}
		gw := &dispatchGateway{}
		steward := agents.NewSmartSteward(unitofwork.NewMemoryStore())
		steward.SetRuntime(gw, g, nil)
		re := rules.NewEngine()
		reg := agents.NewRegistry()
		loop := learning.NewLearningLoop(re, reg)
		loop.SpecialistAfter = 2
		loop.ReplayGate = gated
		if _, err := reg.Save(agents.AgentVersion{Name: "generic", Role: "worker", ModelTier: "cheap", Config: map[string]string{"instructions": "Use routing evidence"}, CreatedBy: "human"}, 0); err != nil {
			t.Fatal(err)
		}
		runner := NewContinuousRunner(steward, agents.NewLLMJudge(gw, g), queue.NewMemoryQueue(), g, re, loop)
		runner.Workers = 1
		runner.Budget.MaxRounds = 2
		err := runner.Run(t.Context(), "Assign the support team")
		if gated {
			if err == nil || gw.focused != 0 {
				t.Fatal("pending rule reached a specialist", err, gw.focused)
			}
			continue
		}
		if err != nil || gw.focused != 2 {
			t.Fatal(err, gw.focused)
		}
		proposals := 0
		for _, entry := range runner.Audit.All() {
			if entry.Action != "proposed" {
				continue
			}
			var m graph.Mutation
			if err := json.Unmarshal(entry.Proposal, &m); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(m.AgentName, "specialist-") || m.AgentRevision != 1 || len(m.RuleRevisions) != 1 {
				t.Fatal(m)
			}
			proposals++
		}
		if proposals != 2 {
			t.Fatal(proposals)
		}
		active := re.ToAgentsRules()
		for _, pattern := range []string{"", "worker failure: provider"} {
			spec, err := loop.WorkerForPattern(pattern, active)
			if err != nil || spec.Name != "generic" {
				t.Fatal(spec, err)
			}
		}
		pattern := "worker failure: format"
		spec, err := loop.WorkerForPattern(pattern, active)
		if err != nil {
			t.Fatal(err)
		}
		def, ok := reg.Get(spec.Name)
		if !ok {
			t.Fatal(spec)
		}
		def.Retired = true
		def.CreatedBy = "human"
		if _, err = reg.Save(def, 1); err != nil {
			t.Fatal(err)
		}
		spec, err = loop.WorkerForPattern(pattern, active)
		if err != nil || spec.Name != "generic" {
			t.Fatal(spec, err)
		}
	}
}

type rejectedShapeGateway struct{ calls int }

func (g *rejectedShapeGateway) Generate(req models.Request) (*models.Response, error) {
	g.calls++
	return &models.Response{Content: `{"wrong_field":true}`}, nil
}
func (g *rejectedShapeGateway) Tokens() int       { return 9 * g.calls }
func (g *rejectedShapeGateway) RecordTool(string) {}

func TestRecordAndRoundBudgetStops(t *testing.T) {
	for _, tc := range []struct {
		name          string
		limit, tokens int
		stop          string
		calls         int
	}{
		{"blocked_after_one_execution", 1, 100, "blocked", 2},
		{"blocked_after_two_executions", 2, 100, "blocked", 4},
		{"budget_exhausted_before_second_round", 3, 18, "token budget", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.NewMemoryGraph()
			g.AddNode(graph.Node{ID: "case", Type: "asset", Props: map[string]any{"condition": "unknown", "status": "open"}})
			gw := &rejectedShapeGateway{}
			steward := agents.NewSmartSteward(unitofwork.NewMemoryStore())
			steward.SetRuntime(gw, g, nil)
			runner := NewContinuousRunner(steward, agents.NewLLMJudge(gw, g), queue.NewMemoryQueue(), g, rules.NewEngine(), nil)
			runner.Workers = 1
			runner.Usage = gw
			runner.Stagnation = 10
			runner.Budget.MaxRecordFailures = tc.limit
			runner.Budget.MaxTokens = tc.tokens
			runner.Budget.MaxRounds = 3
			err := runner.Run(t.Context(), "Edit only condition")
			if err == nil || !strings.Contains(err.Error(), tc.stop) || gw.calls != tc.calls {
				t.Fatal(err, gw.calls)
			}
			record, err := g.GetNode("case")
			if err != nil || record.Props["status"] != "open" || record.Props["condition"] != "unknown" {
				t.Fatal(record, err)
			}
		})
	}
}
