package store

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/llm"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
	"os"
	"testing"
)

func TestUnitStorageParity(t *testing.T) {
	for _, backend := range []string{"memory", "sqlite", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			var st Store
			switch backend {
			case "memory":
				st = NewMemory()
			case "sqlite":
				st = mustSQLite(t)
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN required")
				}
				pg, err := NewPostgresStore("postgres", dsn)
				if err != nil {
					t.Fatal(err)
				}
				st = pg
				t.Cleanup(func() { pg.Close() })
			}
			id := uuid.NewString()
			u := unitofwork.UnitOfWork{ID: id, GoalID: "g", Description: "review", InputSubgraph: "r", Budget: map[string]any{"limit": json.Number("9007199254740993")}, Evidence: []string{"source"}, Images: []llm.ImagePart{{Base64: "fixture"}}, AllowedFields: []string{"quantity"}, FieldTypes: map[string]string{"quantity": "integer"}, ClaimedBy: "steward", Status: unitofwork.StatusClaimed}
			if err := st.CreateUnit(&u); err != nil {
				t.Fatal(err)
			}
			u.Evidence[0] = "changed"
			m := &graph.Mutation{AgentRevision: 3, RuleRevisions: map[string]int{"review": 7}, Evidence: []string{"reason"}}
			if err := st.AttachProposal(id, m); err != nil {
				t.Fatal(err)
			}
			m.AgentRevision = 5
			if err := st.UpdateUnitStatus(id, unitofwork.StatusReviewed); err != nil {
				t.Fatal(err)
			}
			got, err := st.GetUnit(id)
			if err != nil {
				t.Fatal(err)
			}
			if got.Proposal == nil || got.Proposal.AgentRevision != 3 || got.Proposal.RuleRevisions["review"] != 7 || got.Status != unitofwork.StatusReviewed || got.Evidence[0] != "source" || len(got.Images) != 0 || got.InputSubgraph != "r" || got.Budget["limit"] != json.Number("9007199254740993") || got.FieldTypes["quantity"] != "integer" {
				t.Fatal(got)
			}
			if err := st.AttachProposal(id, nil); err != nil {
				t.Fatal(err)
			}
			got, err = st.GetUnit(id)
			if err != nil || got.Proposal != nil {
				t.Fatal(got, err)
			}
			if err := st.UpdateUnitStatus(uuid.NewString(), unitofwork.StatusRejected); err == nil {
				t.Fatal("missing update succeeded")
			}
		})
	}
}
func TestUnitBeforePayloadMigration(t *testing.T) {
	db := mustSQLite(t).(*SQLStore)
	_, err := db.db.Exec("INSERT INTO units(id,goal_id,description,status,proposal) VALUES ('old','g','legacy','claimed','')")
	if err != nil {
		t.Fatal(err)
	}
	u, err := db.GetUnit("old")
	if err != nil || u.Description != "legacy" || u.Status != unitofwork.StatusClaimed || u.Proposal != nil {
		t.Fatal(u, err)
	}
}
