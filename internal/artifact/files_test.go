package artifact

import (
	"errors"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/store"
	"os"
	"strings"
	"testing"
)

func TestPinnedChatFiles(t *testing.T) {
	t.Run("memory", func(t *testing.T) { checkRevisions(t, revision.NewMemory()) })
	t.Run("postgres", func(t *testing.T) {
		dsn := os.Getenv("TEST_POSTGRES_DSN")
		if dsn == "" {
			t.Skip("TEST_POSTGRES_DSN required")
		}
		st, err := store.NewPostgresStore("postgres", dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer st.Close()
		checkRevisions(t, st.Definitions())
	})
}

func checkRevisions(t *testing.T, s revision.Store) {
	t.Helper()
	first, err := Save(s, Ref{}, "brief.md", []byte("initial evidence"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Save(s, first, "brief.md", []byte("corrected evidence"))
	if err != nil || second.ID != first.ID || second.Version != 2 {
		t.Fatal(second, err)
	}
	old, err := Read(s, []Ref{first})
	if err != nil || !strings.Contains(old.Text, "initial evidence") || strings.Contains(old.Text, "corrected evidence") {
		t.Fatal(old, err)
	}
	if _, err = Save(s, first, "brief.md", []byte("stale")); !errors.Is(err, revision.ErrConflict) {
		t.Fatal(err)
	}
	if _, err = Read(s, []Ref{first, second}); err == nil {
		t.Fatal("accepted ambiguous references")
	}
	if _, err = Save(s, Ref{}, "brief.pdf", []byte("unread content")); err == nil {
		t.Fatal("accepted unsupported format")
	}
}
