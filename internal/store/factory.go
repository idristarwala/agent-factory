package store

import (
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"
	_ "github.com/lib/pq"
	"os"
	"path/filepath"
)

// NewFromEnv opens the factory's database. API and runner processes using
// the same database can resolve the same immutable definitions.
func NewFromEnv() (Store, error) {
	st, _, err := OpenFromEnv()
	return st, err
}

// OpenFromEnv also returns the selected backend name for health and readiness.
func OpenFromEnv() (Store, string, error) {
	mode := os.Getenv("FACTORY_STORE_MODE")
	dsn := os.Getenv("DATABASE_URL")
	if mode == "" {
		mode = "sqlite"
		if dsn != "" {
			mode = "postgres"
		}
	}
	switch mode {
	case "memory":
		return NewMemory(), mode, nil
	case "postgres":
		if dsn == "" {
			return nil, mode, fmt.Errorf("postgres requires DATABASE_URL")
		}
		db, err := NewPostgresStore("postgres", dsn)
		if err != nil {
			return nil, mode, fmt.Errorf("cannot open postgres store; check connectivity, credentials and schema permissions")
		}
		return db, mode, nil
	case "sqlite":
		db, err := openLocalSQLite()
		if err != nil {
			return nil, mode, err
		}
		return db, mode, nil
	default:
		return nil, "", fmt.Errorf("unsupported FACTORY_STORE_MODE; choose memory, sqlite or postgres")
	}
}

func openLocalSQLite() (Store, error) {
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
