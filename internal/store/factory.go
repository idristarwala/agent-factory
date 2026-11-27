package store

import (
	"github.com/idristarwala/agent-factory/internal/revision"
	_ "github.com/lib/pq"
	"os"
	"path/filepath"
)

// NewFromEnv opens the factory's database. API and runner processes using
// the same database can resolve the same immutable definitions.
func NewFromEnv() (Store, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return NewPostgresStore("postgres", dsn)
	}
	path := os.Getenv("FACTORY_DB_PATH")
	if path == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(dir, "agent-factory")
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
		path = filepath.Join(dir, "factory.sqlite")
	}
	return NewSQLStore(path)
}
func NewMemory() Store { return NewMemoryStore() }

func FactoryDefinitions(st Store, fallback string) revision.Store {
	name := os.Getenv("FACTORY_ID")
	if name == "" {
		name = fallback
	}
	return revision.Scope{Store: st.Definitions(), Name: name}
}
