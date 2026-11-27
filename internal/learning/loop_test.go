package learning_test

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/learning"
	"github.com/idristarwala/agent-factory/internal/rules"
)

func TestCrossRoundRejectionBecomesRule(t *testing.T) {
	re := rules.NewEngine()
	reg := agents.NewRegistry()
	ll := learning.NewLearningLoop(re, reg)
	ll.ReplayGate = false

	d1 := []agents.Decision{{Approved: false, Reason: "title quality is poor"}}
	gaps1 := ll.Review(d1, nil)
	if len(gaps1) != 0 {
		t.Fatalf("expected no gap after one rejection, got %d", len(gaps1))
	}

	d2 := []agents.Decision{{Approved: false, Reason: "title quality still bad"}}
	gaps2 := ll.Review(d2, nil)
	if len(gaps2) == 0 {
		t.Fatal("expected gap after repeated rejection across rounds")
	}

	updates := ll.ProposeUpdates(gaps2)
	if len(updates) == 0 {
		t.Fatal("expected an update")
	}

	applied, err := ll.Apply(updates)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) == 0 {
		t.Fatal("expected a rule to be applied")
	}

	active := re.Active()
	if len(active) == 0 {
		t.Fatal("expected an active rule")
	}
}

func TestReplayGateHolds(t *testing.T) {
	re := rules.NewEngine()
	reg := agents.NewRegistry()
	ll := learning.NewLearningLoop(re, reg)
	ll.ReplayGate = true

	d := []agents.Decision{
		{Approved: false, Reason: "title quality is poor"},
		{Approved: false, Reason: "title quality is poor"},
	}
	gaps := ll.Review(d, nil)
	if len(gaps) == 0 {
		t.Fatal("expected gaps")
	}
	updates := ll.ProposeUpdates(gaps)
	applied, err := ll.Apply(updates)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) == 0 {
		t.Fatal("expected applied entry")
	}

	active := re.Active()
	if len(active) != 0 {
		t.Fatal("ReplayGate should hold the rule")
	}
}

func TestWorkerFailuresUseSeparateCausesAndKeepHumanRevisions(t *testing.T) {
	re := rules.NewEngine()
	loop := learning.NewLearningLoop(re, agents.NewRegistry())
	failures := []string{"worker validation: invalid field x", "worker validation: invalid field y", "timeout", "json: unexpected EOF"}
	updates := loop.ProposeUpdates(loop.Review(nil, failures))
	applied, err := loop.Apply(updates)
	if err != nil || len(applied) != 3 {
		t.Fatal(applied, err)
	}
	original := re.Active()[0]
	human := original
	human.CreatedBy = "human"
	human.Description = "operator's revised guidance"
	if _, err = re.Save(human, 1); err != nil {
		t.Fatal(err)
	}
	repeated, err := loop.Apply(updates)
	if err != nil || len(repeated) != 0 {
		t.Fatal(repeated, err)
	}
	history, err := re.List(original.ID)
	if err != nil || len(history) != 2 || history[1].Description != human.Description {
		t.Fatal(history, err)
	}
}
