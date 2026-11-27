package factory

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/artifact"
	"github.com/idristarwala/agent-factory/internal/obs"
	"github.com/idristarwala/agent-factory/internal/settings"
	"github.com/idristarwala/agent-factory/internal/store"
)

func TestDurableReviewEvidence(t *testing.T) {
	for _, mode := range []string{"sqlite", "postgres"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "postgres" && os.Getenv("TEST_POSTGRES_DSN") == "" {
				t.Skip("TEST_POSTGRES_DSN required")
			}
			t.Setenv("FACTORY_STORE_MODE", mode)
			t.Setenv("DATABASE_URL", os.Getenv("TEST_POSTGRES_DSN"))
			t.Setenv("FACTORY_DB_PATH", filepath.Join(t.TempDir(), "workspace.sqlite"))
			t.Setenv("FACTORY_ID", uuid.NewString())
			open := func() store.Store {
				t.Helper()
				db, err := store.NewFromEnv()
				if err != nil {
					t.Fatal(err)
				}
				return db
			}
			db := open()
			// Close the current handle even if setup or a check fails.
			defer func() { db.Close() }()
			ws := Workspace{Store: store.FactoryDefinitions(db, "")}
			first := []byte("Original evidence\r\n")
			second := []byte("Reviewed evidence — keep this line\n")
			ref, err := artifact.Save(ws.Store, artifact.Ref{}, "source.txt", first)
			if err != nil {
				t.Fatal(err)
			}
			next, err := artifact.Save(ws.Store, ref, "source.txt", second)
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC()
			audit := []obs.AuditEntry{
				{TraceID: "unit-1", EntityID: ref.ID, Action: "proposed", Actor: "worker", Timestamp: now},
				{TraceID: "unit-1", EntityID: ref.ID, Action: "approved", Actor: "reviewer", Feedback: "verified source", Timestamp: now},
			}
			run, err := ws.FinishContinuous("Review evidence", now, settings.Settings{}, 1, &ContinuousUsage{}, audit, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			db = open()
			ws.Store = store.FactoryDefinitions(db, "")
			saved, err := ws.Get(run.ID)
			if err != nil || !reflect.DeepEqual(saved.Audit, audit) {
				t.Fatalf("audit after restart: %+v, %v", saved.Audit, err)
			}
			for pin, want := range map[artifact.Ref][]byte{ref: first, next: second} {
				f, err := artifact.ReadFile(ws.Store, pin)
				if err != nil || !bytes.Equal(f.Data, want) || f.SHA256 != fmt.Sprintf("%x", sha256.Sum256(want)) {
					t.Fatalf("artifact revision %d after restart: %+v, %v", pin.Version, f, err)
				}
			}
		})
	}
}
