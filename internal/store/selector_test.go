package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/revision"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWorkspaceStore(t *testing.T) {
	for _, c := range []struct {
		mode, dsn, want string
		valid           bool
	}{
		{"", "", "sqlite", true}, {"memory", "malformed", "memory", true}, {"sqlite", "malformed", "sqlite", true}, {"postgres", "", "postgres", false}, {"", "malformed", "postgres", false}, {"unsupported", "", "", false},
	} {
		t.Run(c.mode+"-"+c.dsn, func(t *testing.T) {
			t.Setenv("FACTORY_STORE_MODE", c.mode)
			t.Setenv("DATABASE_URL", c.dsn)
			t.Setenv("FACTORY_DB_PATH", filepath.Join(t.TempDir(), "workspace.db"))
			db, mode, err := OpenFromEnv()
			if mode != c.want || (err == nil) != c.valid {
				t.Fatalf("mode=%q err=%v", mode, err)
			}
			if db != nil {
				defer db.Close()
			}
		})
	}
}

func TestIndependentRevisionWriters(t *testing.T) {
	for _, mode := range []string{"sqlite", "postgres"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "postgres" && os.Getenv("TEST_POSTGRES_DSN") == "" {
				t.Skip("requires TEST_POSTGRES_DSN")
			}
			t.Setenv("FACTORY_STORE_MODE", mode)
			t.Setenv("DATABASE_URL", os.Getenv("TEST_POSTGRES_DSN"))
			t.Setenv("FACTORY_DB_PATH", filepath.Join(t.TempDir(), "race.db"))
			id := uuid.NewString()
			open := func() Store {
				t.Helper()
				s, err := NewFromEnv()
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { s.Close() })
				return s
			}
			writers := []Store{open(), open()}
			for expected := 0; expected < 2; expected++ {
				start := make(chan struct{})
				results := make(chan error, 2)
				for i, db := range writers {
					go func() {
						<-start
						_, err := revision.SaveDocument(db.Definitions(), id, expected, "human", json.RawMessage(fmt.Sprintf(`{"round":%d,"writer":%d}`, expected, i)))
						results <- err
					}()
				}
				close(start)
				first, second := <-results, <-results
				if !(first == nil && errors.Is(second, revision.ErrConflict) || second == nil && errors.Is(first, revision.ErrConflict)) {
					t.Fatalf("race at %d: %v, %v", expected, first, second)
				}
			}
			for _, db := range writers {
				db.Close()
			}
			rows, err := revision.Documents(open().Definitions(), id)
			if err != nil || len(rows) != 2 || rows[0].Version != 1 || rows[1].Version != 2 {
				t.Fatal(rows, err)
			}
		})
	}
}
