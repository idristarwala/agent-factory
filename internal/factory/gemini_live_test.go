//go:build gemini_live

package factory

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/schema"
)

func TestLiveGeminiFactoryRun(t *testing.T) {
	gateway, err := models.NewGeminiFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	def, err := os.ReadFile("../../examples/service-desk/factory.json")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../examples/service-desk/records.json")
	if err != nil {
		t.Fatal(err)
	}
	var records []schema.Entity
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	ws := Workspace{Store: revision.NewMemory(), Gateway: gateway}
	if _, err := revision.SaveDocument(ws.Store, "factory/desk", 0, "human", def); err != nil {
		t.Fatal(err)
	}
	run, err := ws.Start(RunRequest{Description: "Assign the support team from the routing guide", Factory: "desk", Records: records})
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.Execute(run.ID); err != nil {
		t.Fatal(err)
	}
	run, err = ws.Get(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "in_review" || len(run.Decisions) != 1 || len(run.Calls) != 2 {
		t.Fatalf("status %s, decisions %d, model calls %d", run.Status, len(run.Decisions), len(run.Calls))
	}
	for _, call := range run.Calls {
		if call.Model == "" || call.Total <= 0 {
			t.Fatal("missing model or token usage")
		}
	}
	run, err = ws.Decide(run.ID, run.Decisions[0].ID, "approved", "", run.Version)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "completed" || run.Records[0].Attributes["assigned_team"] != "Workplace IT" {
		t.Fatalf("status %s, assigned team %v", run.Status, run.Records[0].Attributes["assigned_team"])
	}
}
