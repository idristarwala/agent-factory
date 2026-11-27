package store

import (
	"bytes"
	"os"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func TestStoreContract(t *testing.T) {
	t.Run("memory", func(t *testing.T) { runContract(t, mustMemory()) })
	t.Run("sqlite", func(t *testing.T) { runContract(t, mustSQLite(t)) })
	t.Run("postgres", func(t *testing.T) {
		dsn := os.Getenv("TEST_POSTGRES_DSN")
		if dsn == "" {
			t.Skip("TEST_POSTGRES_DSN required")
		}
		st, err := NewPostgresStore("postgres", dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { st.Close() })
		runContract(t, st)
	})
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

	prefix := uuid.NewString()
	nodeID := graph.NodeID(prefix + "-node")
	addedID := graph.NodeID(prefix + "-added")
	unitID := prefix + "-unit"

	// Node round-trip
	n := graph.Node{ID: nodeID, Type: "record"}
	if err := st.UpsertNode(n); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	got, err := st.GetNode(nodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.Type != "record" {
		t.Errorf("type = %q", got.Type)
	}

	// Mutation
	m := &graph.Mutation{
		Adds: []graph.Node{{ID: addedID, Type: "related"}},
	}
	if err := st.ProposeMutation(m); err != nil {
		t.Fatalf("ProposeMutation: %v", err)
	}
	if _, err := st.GetNode(addedID); err != nil {
		t.Fatalf("node from mutation missing: %v", err)
	}

	// Unit of work
	u := &unitofwork.UnitOfWork{ID: unitID, GoalID: "g1", Description: "review record", Status: unitofwork.StatusProposed}
	if err := st.CreateUnit(u); err != nil {
		t.Fatalf("CreateUnit: %v", err)
	}
	ug, err := st.GetUnit(unitID)
	if err != nil {
		t.Fatalf("GetUnit: %v", err)
	}
	if ug.Status != unitofwork.StatusProposed {
		t.Errorf("status = %s", ug.Status)
	}
	if err := st.UpdateUnitStatus(unitID, unitofwork.StatusClaimed); err != nil {
		t.Fatalf("UpdateUnitStatus: %v", err)
	}
	ug, _ = st.GetUnit(unitID)
	if ug.Status != unitofwork.StatusClaimed {
		t.Errorf("updated status = %s", ug.Status)
	}

	// Rules
	rule := []byte(`{"kind":"review","id":"` + prefix + `"}`)
	if err := st.SaveRuleVersion(prefix, 1, rule); err != nil {
		t.Fatalf("SaveRuleVersion: %v", err)
	}
	rules, err := st.ActiveRules()
	if err != nil {
		t.Fatalf("ActiveRules: %v", err)
	}
	if !slices.ContainsFunc(rules, func(data []byte) bool { return bytes.Equal(data, rule) }) {
		t.Error("expected at least one rule")
	}

	// Entries
	if err := st.SaveEntry(prefix, "retain source", "judge", nil); err != nil {
		t.Fatalf("SaveEntry: %v", err)
	}
	ents, err := st.EntriesByKind(prefix)
	if err != nil {
		t.Fatalf("EntriesByKind: %v", err)
	}
	if len(ents) != 1 || string(ents[0]) != "retain source" {
		t.Error("expected at least one entry")
	}
}
