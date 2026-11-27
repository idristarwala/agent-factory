package store

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/graph"
)

func TestNodeStorageParity(t *testing.T) {
	for _, backend := range []string{"memory", "sqlite", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			var s Store
			switch backend {
			case "memory":
				s = NewMemory()
			case "sqlite":
				s = mustSQLite(t)
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN required")
				}
				pg, err := NewPostgresStore("postgres", dsn)
				if err != nil {
					t.Fatal(err)
				}
				s = pg
				t.Cleanup(func() { pg.Close() })
			}
			id := graph.NodeID(uuid.NewString())
			n := graph.Node{ID: id, Type: "record", Props: map[string]any{"nested": map[string]any{"id": json.Number("9007199254740993")}}}
			if err := s.UpsertNode(n); err != nil {
				t.Fatal(err)
			}
			delete(n.Props, "nested")
			check := func() {
				t.Helper()
				got, err := s.GetNode(id)
				if err != nil {
					t.Fatal(err)
				}
				b, err := json.Marshal(got.Props)
				if err != nil || string(b) != `{"nested":{"id":9007199254740993}}` {
					t.Fatal(string(b), err)
				}
				delete(got.Props, "nested")
			}
			check()
			check()
			err := s.ProposeMutation(&graph.Mutation{Updates: []graph.Node{n}, Adds: []graph.Node{{ID: "bad", Props: map[string]any{"v": make(chan int)}}}})
			if err == nil {
				t.Fatal("expected non-JSON value to fail")
			}
			check()
		})
	}
}
func TestLegacyEmptyProperties(t *testing.T) {
	s := mustSQLite(t).(*SQLStore)
	if _, err := s.db.Exec("INSERT INTO nodes(id,type,props) VALUES ('old','record',''),('nullable','record',NULL)"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []graph.NodeID{"old", "nullable"} {
		n, err := s.GetNode(id)
		if err != nil || n.Type != "record" || n.Props != nil {
			t.Fatal(n, err)
		}
	}
}
