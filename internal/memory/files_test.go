package memory_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/memory"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/store"
)

func TestFileBackends(t *testing.T) {
	for _, backend := range []string{"memory", "sqlite", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			dbPath := t.TempDir() + "/store.sqlite"
			open := func() store.Store {
				switch backend {
				case "memory":
					t.Setenv("FACTORY_STORE_MODE", "memory")
					t.Setenv("DATABASE_URL", "invalid")
					s, err := store.NewFromEnv()
					if err != nil {
						t.Fatal(err)
					}
					return s
				case "sqlite":
					s, err := store.NewSQLStore(dbPath)
					if err != nil {
						t.Fatal(err)
					}
					return s
				default:
					dsn := os.Getenv("TEST_POSTGRES_DSN")
					if dsn == "" {
						t.Skip("TEST_POSTGRES_DSN required")
					}
					s, err := store.NewPostgresStore("postgres", dsn)
					if err != nil {
						t.Fatal(err)
					}
					return s
				}
			}
			s := open()
			defer func() { s.Close() }()
			files := memory.Files{Store: s.Definitions(), Factory: uuid.NewString()}
			const path = "notes/routes.txt"
			if _, err := files.Write(path, "first", "human", 0); err != nil {
				t.Fatal(err)
			}
			if _, err := files.Write(path, "second", "steward", 1); err != nil {
				t.Fatal(err)
			}
			if _, err := files.Write(path, "stale", "human", 1); !errors.Is(err, revision.ErrConflict) {
				t.Fatal(err)
			}
			other := memory.Files{Store: s.Definitions(), Factory: files.Factory + "2"}
			if _, err := other.Read(memory.Ref{Path: path}); !errors.Is(err, fs.ErrNotExist) {
				t.Fatal(err)
			}
			if _, err := files.Write("../hidden", "x", "human", 0); !errors.Is(err, memory.ErrInvalid) {
				t.Fatal(err)
			}
			refs, err := files.List("notes")
			if err != nil || len(refs) != 1 || refs[0].Version != 2 {
				t.Fatal(refs, err)
			}
			s.Close()
			s = open()
			files.Store = s.Definitions()
			first, err := files.Read(memory.Ref{Path: path, Version: 1})
			if backend == "memory" {
				if !errors.Is(err, fs.ErrNotExist) {
					t.Fatal(err)
				}
				return
			}
			if err != nil || first.Content != "first" || first.Author != "human" {
				t.Fatal(first, err)
			}
			last, err := files.Read(memory.Ref{Path: path})
			if err != nil || last.Version != 2 || last.Author != "steward" {
				t.Fatal(last, err)
			}
		})
	}
}
