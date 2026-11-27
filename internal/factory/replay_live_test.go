//go:build openrouter_live

package factory

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/store"
	"os"
	"path/filepath"
	"testing"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/configuredgateway"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/schema"
)

// An independently authored expected label is approved before either live evaluation.
func TestLiveReviewedReplay(t *testing.T) {
	gw, err := configuredgateway.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FACTORY_DB_PATH", filepath.Join(t.TempDir(), "replay.sqlite"))
	t.Setenv("FACTORY_ID", "live-replay-"+uuid.NewString())
	if os.Getenv("TEST_POSTGRES_DSN") != "" {
		t.Setenv("FACTORY_STORE_MODE", "postgres")
		t.Setenv("DATABASE_URL", os.Getenv("TEST_POSTGRES_DSN"))
	} else {
		t.Setenv("FACTORY_STORE_MODE", "sqlite")
	}
	db, err := store.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	st := store.FactoryDefinitions(db, "")
	_, err = revision.SaveDocument(st, "factory/desk", 0, "human", []byte(`{"record_type":"ticket","fields":{"team":"string"}}`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = agents.NewStoredRegistry(st).Save(agents.AgentVersion{Name: "judge", Role: "judge", ModelTier: "frontier", Config: map[string]string{"max_output_tokens": "1500"}, CreatedBy: "human"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	ws := Workspace{Store: st, Gateway: approvedFixture{}}
	run, err := ws.Start(RunRequest{Factory: "desk", Description: "Set team to the exact string support, as required by the approved routing guide.", Records: []schema.Entity{{ID: "case-1", Type: "ticket", Attributes: map[string]any{"team": "unassigned", "routing_guide": "Set team to support."}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.Execute(run.ID); err != nil {
		t.Fatal(err)
	}
	run, err = ws.Get(run.ID)
	if err != nil || len(run.Decisions) != 1 {
		t.Fatal(err)
	}
	run, err = ws.Decide(run.ID, run.Decisions[0].ID, "approved", "fixture label", run.Version)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	ws.Store = store.FactoryDefinitions(reopened, "")
	usage := &ContinuousUsage{Gateway: gw}
	ws.Gateway = usage
	replay, err := ws.ReviewedReplay(fmt.Sprintf("%s@%d", run.ID, run.Version), graph.NewMemoryGraph(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := replay(rules.VersionedRule{ID: "routing-guide", Version: 1, Description: "Preserve exact team names from the approved routing guide."}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Cases) != 1 || len(usage.calls) != 4 || usage.Tokens() <= 0 {
		t.Fatal(report, usage.calls)
	}
	for _, c := range usage.calls {
		t.Logf("%s %s %s %d tokens", c.UnitID, c.Role, c.Model, c.Total)
	}
	t.Logf("reviewed outcomes %+v", report.Cases)
}
