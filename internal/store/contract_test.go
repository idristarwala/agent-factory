package store

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// TestStoreContract runs the shared contract against every Store implementation
// that can be constructed without external services (Memory + SQLite SQLStore).
// PostgresStore is covered by the same methods; run it against a real instance
// with NewPostgresStore when a DSN is available.
func TestStoreContract(t *testing.T) {
	stores := map[string]Store{
		"memory": mustMemory(),
		"sqlite": mustSQLite(t),
	}

	for name, st := range stores {
		st := st
		t.Run(name, func(t *testing.T) {
			runContract(t, st)
		})
	}
}

func mustMemory() Store {
	return NewMemoryStore()
}

func mustSQLite(t *testing.T) Store {
	t.Helper()
	s, err := NewSQLStore(":memory:")
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func runContract(t *testing.T, st Store) {
	t.Helper()

	// Node round-trip
	n := graph.Node{ID: "n1", Type: "product"}
	if err := st.UpsertNode(n); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	got, err := st.GetNode("n1")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.Type != "product" {
		t.Errorf("type = %q", got.Type)
	}

	// Mutation
	m := &graph.Mutation{
		Adds: []graph.Node{{ID: "n2", Type: "variant"}},
	}
	if err := st.ProposeMutation(m); err != nil {
		t.Fatalf("ProposeMutation: %v", err)
	}
	if _, err := st.GetNode("n2"); err != nil {
		t.Fatalf("node from mutation missing: %v", err)
	}

	// Unit of work
	u := &unitofwork.UnitOfWork{ID: "u1", GoalID: "g1", Description: "fix titles", Status: unitofwork.StatusProposed}
	if err := st.CreateUnit(u); err != nil {
		t.Fatalf("CreateUnit: %v", err)
	}
	ug, err := st.GetUnit("u1")
	if err != nil {
		t.Fatalf("GetUnit: %v", err)
	}
	if ug.Status != unitofwork.StatusProposed {
		t.Errorf("status = %s", ug.Status)
	}
	if err := st.UpdateUnitStatus("u1", unitofwork.StatusClaimed); err != nil {
		t.Fatalf("UpdateUnitStatus: %v", err)
	}
	ug, _ = st.GetUnit("u1")
	if ug.Status != unitofwork.StatusClaimed {
		t.Errorf("updated status = %s", ug.Status)
	}

	// Rules
	if err := st.SaveRuleVersion("r1", 1, []byte(`{"kind":"title"}`)); err != nil {
		t.Fatalf("SaveRuleVersion: %v", err)
	}
	rules, err := st.ActiveRules()
	if err != nil {
		t.Fatalf("ActiveRules: %v", err)
	}
	if len(rules) == 0 {
		t.Error("expected at least one rule")
	}

	// Entries
	if err := st.SaveEntry("learning", "prefer short titles", "judge", nil); err != nil {
		t.Fatalf("SaveEntry: %v", err)
	}
	ents, err := st.EntriesByKind("learning")
	if err != nil {
		t.Fatalf("EntriesByKind: %v", err)
	}
	if len(ents) == 0 {
		t.Error("expected at least one entry")
	}
}
