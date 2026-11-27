package rules

import (
	"reflect"
	"testing"
)

func TestInvalidActivationPreservesRuleHistory(t *testing.T) {
	e := NewEngine()
	first := e.Propose(VersionedRule{Description: "first", Active: true})
	next := e.Propose(VersionedRule{ID: first.ID, Description: "second"})
	before := e.History(first.ID)
	if err := e.Activate(first.ID, next.Version+1); err == nil {
		t.Fatal("expected invalid version error")
	}
	if !reflect.DeepEqual(before, e.History(first.ID)) || !reflect.DeepEqual(e.Active(), []VersionedRule{first}) {
		t.Fatal("invalid activation changed rule state")
	}
	if err := e.Activate(first.ID, next.Version); err != nil {
		t.Fatal(err)
	}
	active := e.Active()
	if len(active) != 1 || active[0].Version != next.Version+1 || active[0].ActivatedFrom != next.Version {
		t.Fatalf("active rules: %+v", active)
	}
}

func TestPromotingHistoricalContentAppendsRollback(t *testing.T) {
	e := NewEngine()
	a := e.Propose(VersionedRule{ID: "r", Description: "original", Active: true})
	b := e.Propose(VersionedRule{ID: "r", Description: "edit", Active: true})
	rollback, err := e.Promote("r", a.Version, b.Version, "human")
	if err != nil || rollback.Version != 3 || rollback.Description != a.Description || rollback.CreatedBy != "human" || rollback.ParentID != "r:2" {
		t.Fatal(rollback, err)
	}
	h := e.History("r")
	if h[0] != a || h[1] != b {
		t.Fatal("rewrote history")
	}
}
