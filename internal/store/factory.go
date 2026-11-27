package store

import (
	"os"
)

// NewFromEnv returns a Store based on environment.
// If DATABASE_URL is set, it opens a Postgres store (driver "postgres" must be
// registered by the caller via blank import of github.com/lib/pq or similar).
// Otherwise it returns an in-memory store.
func NewFromEnv() (Store, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return NewMemory(), nil
	}
	return NewPostgresStore("postgres", dsn)
}

// NewMemory is a convenience alias.
func NewMemory() Store {
	return NewMemoryStore()
}
