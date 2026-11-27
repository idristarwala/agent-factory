package store

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/idristarwala/agent-factory/internal/revision"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func TestSQLiteContentionAndStaleRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "factory.sqlite")
	open := func() *SQLStore {
		t.Helper()
		s, err := NewSQLStore(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		return s
	}
	first, second := open(), open()
	// Force the impatient caller to observe the real lock without waiting five seconds.
	if _, err := second.db.Exec("PRAGMA busy_timeout=0"); err != nil {
		t.Fatal(err)
	}
	tx, err := first.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec("INSERT INTO config_versions VALUES ('aborted',1,'draft')"); err != nil {
		t.Fatal(err)
	}
	snapshot := revision.Snapshot{ID: "definition", Version: 1, Data: []byte("first")}
	insert := func() error { return second.Definitions().Insert(revision.Config, snapshot) }
	var busy *sqlite.Error
	if err := insert(); !errors.As(err, &busy) || busy.Code() != sqlite3.SQLITE_BUSY {
		t.Fatalf("expected SQLITE_BUSY while the writer holds the lock, got %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := insert(); err != nil {
		t.Fatalf("retry after rollback: %v", err)
	}
	snapshot.Data = []byte("replacement")
	if err := insert(); !errors.Is(err, revision.ErrConflict) {
		t.Fatalf("expected stale revision after commit, got %v", err)
	}
	history, err := first.Definitions().Read(revision.Config, "")
	if err != nil || len(history) != 1 || string(history[0].Data) != "first" || history[0].ID != snapshot.ID {
		t.Fatalf("unexpected committed history: %+v, %v", history, err)
	}
}

func TestSelectedSQLiteTimeoutSurvivesReconnect(t *testing.T) {
	t.Setenv("FACTORY_STORE_MODE", "sqlite")
	t.Setenv("FACTORY_DB_PATH", filepath.Join(t.TempDir(), "store.sqlite"))
	for _, raw := range []string{"0", "23", "2147483647"} {
		t.Setenv("FACTORY_SQLITE_BUSY_TIMEOUT_MS", raw)
		st, _, err := OpenFromEnv()
		if err != nil {
			t.Fatal(err)
		}
		db := st.(*SQLStore).db
		db.SetMaxIdleConns(0)
		for i := 0; i < 2; i++ {
			var got string
			if err := db.QueryRow("PRAGMA busy_timeout").Scan(&got); err != nil || got != raw {
				t.Fatalf("setting %s, connection %d: got %s, error %v", raw, i, got, err)
			}
		}
		st.Close()
	}
	for _, raw := range []string{"", "-1", "5s", "2147483648"} {
		t.Setenv("FACTORY_SQLITE_BUSY_TIMEOUT_MS", raw)
		if _, _, err := OpenFromEnv(); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	t.Setenv("FACTORY_STORE_MODE", "memory")
	st, _, err := OpenFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
}
