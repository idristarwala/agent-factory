package factory

import (
	"fmt"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/memory"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/schema"
)

type memoryCheck struct{ calls []string }

func (m *memoryCheck) Generate(req models.Request) (*models.Response, error) {
	if !strings.Contains(req.User, "REFERENCE-317") || strings.Contains(req.User, "LATER-128") {
		return nil, fmt.Errorf("pinned evidence missing from %s", req.CallerRole)
	}
	m.calls = append(m.calls, req.CallerRole)
	return scripted{}.Generate(req)
}
func TestPinnedMemoryInWorkerAndJudge(t *testing.T) {
	s := revision.NewMemory()
	if _, err := revision.SaveDocument(s, "factory/desk", 0, "human", []byte(`{"record_type":"request","fields":{"priority":"integer"}}`)); err != nil {
		t.Fatal(err)
	}
	files := memory.Files{Store: s, Factory: "desk"}
	path := "notes/priority.txt"
	if _, err := files.Write(path, "REFERENCE-317", "human", 0); err != nil {
		t.Fatal(err)
	}
	gateway := &memoryCheck{}
	ws := Workspace{Store: s, Gateway: gateway}
	r, err := ws.Start(RunRequest{Description: "triage", Factory: "desk", Records: []schema.Entity{{ID: "R1", Type: "request", Attributes: map[string]any{"priority": 1}}}, Memory: []memory.Ref{{Path: path}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := files.Write(path, "LATER-128", "steward", 1); err != nil {
		t.Fatal(err)
	}
	if err = ws.Execute(r.ID); err != nil {
		t.Fatal(err)
	}
	r, err = ws.Get(r.ID)
	if err != nil || len(r.Decisions) != 1 || strings.Join(gateway.calls, ",") != "worker,judge" {
		t.Fatal(r, err)
	}
	d := r.Decisions[0]
	for i := 0; i < 2; i++ {
		if _, err = ws.Decide(r.ID, d.ID, "approved", "Useful for future triage", r.Version); err != nil {
			t.Fatal(err)
		}
	}
	f, err := files.Read(memory.Ref{Path: "reviews/" + d.ID + ".txt"})
	if err != nil || f.Version != 1 || f.Content != "Useful for future triage" {
		t.Fatal(f, err)
	}
}
