package swarm

import (
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
	applied := loop.Apply(updates)

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
		t.Fatal("expected a gap after repeated rejections across rounds")
	}
}

func TestTitleTaskSchemaOnlyAllowsTitleFields(t *testing.T) {
	allowed := []string{"title", "description"}
	required := []string{"title"}
	schema := agents.BuildResponseSchema(allowed, required)

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing properties")
	}
	if _, has := props["size"]; has {
		t.Error("title task schema should not include size")
	}
	if _, has := props["color"]; has {
		t.Error("title task schema should not include color")
	}
}

func TestLearnedRuleReachesWorkerNextRound(t *testing.T) {
	re := rules.NewEngine()
	loop := learning.NewLearningLoop(re, agents.NewRegistry())

	gaps := loop.Review(nil, []string{"format failure"})
	updates := loop.ProposeUpdates(gaps)
	loop.Apply(updates)

	active := re.ToAgentsRules()
	if len(active) == 0 {
		t.Fatal("expected an active rule for the next round")
	}
}
