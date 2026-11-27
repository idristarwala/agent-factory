package factory

import (
	"errors"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/schema"
	"github.com/idristarwala/agent-factory/internal/store"
	"path/filepath"
	"strings"
	"testing"
)

type scripted struct{}

func (scripted) Generate(req models.Request) (*models.Response, error) {
	response := "APPROVE: supported by source"
	if req.Schema != nil {
		response = `{"priority":3}`
		if f := req.Schema["properties"].(map[string]any); len(f) != 1 || f["priority"].(map[string]any)["type"] != "integer" {
			return nil, errors.New("incorrect service schema")
		}
	}
	return &models.Response{Content: response, Model: "test", Usage: map[string]int{"prompt_tokens": 10, "completion_tokens": 2, "total_tokens": 12}}, nil
}
func TestPinnedNonRetailRunSurvivesRestartAndReview(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.sqlite")
	db, err := store.NewSQLStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s := db.Definitions()
	_, err = revision.SaveDocument(s, "factory/desk", 0, "human", []byte(`{"record_type":"request","fields":{"priority":"integer"}}`))
	if err != nil {
		t.Fatal(err)
	}
	ws := Workspace{Store: s, Gateway: scripted{}, Prices: map[string]Price{"test": {1, 2}}}
	r, err := ws.Start(RunRequest{Description: "Set priority from evidence", Factory: "desk", Records: []schema.Entity{{ID: "req1", Type: "request", Attributes: map[string]any{"priority": 1, "status": "open"}}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = revision.SaveDocument(s, "factory/desk", 1, "human", []byte(`{"record_type":"other","fields":{"owner":"string"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err = ws.Execute(r.ID); err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = store.NewSQLStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ws.Store = db.Definitions()
	r, err = ws.Get(r.ID)
	if err != nil || r.Status != "in_review" || len(r.Calls) != 2 || r.Calls[0].Cost <= 0 || r.Plan.Revision != 1 || r.Plan.Worker.Revision != 1 {
		t.Fatal(r, err)
	}
	d := r.Decisions[0]
	if !strings.Contains(d.Reason, "supported") {
		t.Fatal(d)
	}
	r, err = ws.Decide(r.ID, d.ID, "approved", "", r.Version)
	if err != nil || r.Records[0].Attributes["priority"] != float64(3) || r.Records[0].Attributes["status"] != "open" || r.Records[0].Type != "request" || r.Status != "completed" {
		t.Fatal(r, err)
	}
	if _, err = ws.Decide(r.ID, d.ID, "rejected", "", r.Version-1); !errors.Is(err, revision.ErrConflict) {
		t.Fatal(err)
	}
}

func TestSelectedRulesPinnedAtStart(t *testing.T) {
	s := revision.NewMemory()
	_, err := revision.SaveDocument(s, "factory/desk", 0, "human", []byte(`{"record_type":"request","fields":{"owner":"string"}}`))
	if err != nil {
		t.Fatal(err)
	}
	engine := rules.NewStoredEngine(s)
	for _, id := range []string{"a", "b"} {
		if _, err := engine.Save(rules.VersionedRule{ID: id, Active: true}, 0); err != nil {
			t.Fatal(err)
		}
	}
	ws := Workspace{Store: s}
	req := RunRequest{Description: "Assign owner", Factory: "desk", Records: []schema.Entity{{ID: "req", Type: "request"}}, SelectedRules: map[string]int{"b": 1}}
	run, err := ws.Start(req)
	if err != nil || len(run.Plan.Rules) != 1 || run.Plan.Rules[0].ID != "b" {
		t.Fatal(run.Plan.Rules, err)
	}
	req.SelectedRules["b"] = 2
	if _, err = ws.Start(req); err == nil {
		t.Fatal("accepted stale selection")
	}
	req.SelectedRules = nil
	run, err = ws.Start(req)
	if err != nil || len(run.Plan.Rules) != 2 {
		t.Fatal(run.Plan.Rules, err)
	}
}

func TestCancelledEvaluationRetainsUsage(t *testing.T) {
	w := Workspace{Store: revision.NewMemory()}
	r, err := w.Start(RunRequest{Description: "review"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Cancel(r.ID); err != nil {
		t.Fatal(err)
	}
	r.Tools = map[string]int{"read_goal_artifacts": 1}
	r.Calls = []Call{{Role: "worker", Total: 12}}
	r.Status = "in_review"
	r.Decisions = []Review{{Status: "pending"}}
	if err = w.checkpointEvaluation(&r); err != nil {
		t.Fatal(err)
	}
	stored, err := w.Get(r.ID)
	if err != nil || stored.Status != "cancelled" || len(stored.Decisions) != 0 || len(stored.Calls) != 1 || stored.Tools["read_goal_artifacts"] != 1 {
		t.Fatal(stored, err)
	}
}
