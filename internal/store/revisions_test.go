package store_test

import (
	"errors"
	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/store"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSQLRevisionHistorySurvivesRestartAndPromotion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "factory.sqlite")
	st, err := store.NewSQLStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = st.SaveRuleVersion("legacy", 1, []byte("old feedback")); err != nil {
		t.Fatal(err)
	}
	if err = st.SaveRuleVersion("legacy", 1, []byte("overwrite")); !errors.Is(err, revision.ErrConflict) {
		t.Fatal(err)
	}
	engine := rules.NewStoredEngine(st.Definitions())
	history, _ := engine.List("legacy")
	if len(history) != 1 || history[0].Description != "old feedback" {
		t.Fatal(history)
	}
	next, err := engine.Save(rules.VersionedRule{ID: "legacy", Description: "preserve source", CreatedBy: "human"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := engine.List("legacy")
	promoted, err := engine.Promote("legacy", next.Version, 2, "reviewer")
	if err != nil || promoted.Version != 3 {
		t.Fatal(promoted, err)
	}
	registry := agents.NewStoredRegistry(st.Definitions())
	if _, err = registry.Save(agents.AgentVersion{Name: "steward", Role: "steward", CreatedBy: "human"}, 0); err != nil {
		t.Fatal(err)
	}
	other := rules.NewStoredEngine(revision.Scope{Store: st.Definitions(), Name: "another"})
	if _, err = other.Save(rules.VersionedRule{ID: "legacy", Description: "private to another factory", Active: true}, 0); err != nil {
		t.Fatal(err)
	}
	st.Close()
	st, err = store.NewSQLStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	engine = rules.NewStoredEngine(revision.Scope{Store: st.Definitions()})
	after, err := engine.List("")
	if err != nil || len(after) != 3 || !reflect.DeepEqual(before, after[:2]) {
		t.Fatal(after, err)
	}
	if v := engine.Active(); len(v) != 1 || v[0].Version != 3 {
		t.Fatal(v)
	}
	registry = agents.NewStoredRegistry(st.Definitions())
	if v, ok := registry.Get("steward"); !ok || v.Revision != 1 {
		t.Fatal(v)
	}
}
