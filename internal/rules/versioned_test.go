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
	if len(active) != 1 || active[0].Version != next.Version {
		t.Fatalf("active rules: %+v", active)
	}
}
