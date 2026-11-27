package factory

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/schema"
)

type proposalJudge struct{ t *testing.T }

func (g proposalJudge) Generate(req models.Request) (*models.Response, error) {
	if strings.Contains(req.User, "private-config-value") {
		g.t.Fatal("transport configuration reached model")
	}
	if req.CallerRole == "judge" {
		for _, needle := range []string{`"side_effect":"irreversible"`, `"changes":{"priority":3}`, `"record":"r1"`, `"tool_revision":1`} {
			if !strings.Contains(req.User, needle) {
				g.t.Errorf("proposal sent to judge missing %s", needle)
			}
		}
	}
	return scripted{}.Generate(req)
}

func TestExternalChangeProposalPinsAuthorizationAndNeverDispatches(t *testing.T) {
	var hits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	defer target.Close()
	ws := Workspace{Store: revision.NewMemory(), Gateway: proposalJudge{t}}
	save := func(key string, expected int, body string) {
		t.Helper()
		if _, err := revision.SaveDocument(ws.Store, key, expected, "human", []byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	save("factory/desk", 0, `{"record_type":"request","fields":{"priority":"integer"},"propose_tool":"update"}`)
	def := fmt.Sprintf(`{"url":%q,"headers":{"X-Key":"private-config-value"},"side_effect":"irreversible","record_argument":"record","changes_argument":"changes"}`, target.URL)
	save("tool/update", 0, def)
	grant := `{"grants":[{"pack":"desk","tools":["update"],"side_effects":["irreversible"]}]}`
	save("pack/desk", 0, strings.Replace(grant, "irreversible", "reversible", 1))
	req := RunRequest{Description: "assign priority", Factory: "desk", Records: []schema.Entity{{ID: "r1", Type: "request", Attributes: map[string]any{"priority": 1}}}}
	if _, err := ws.Start(req); err == nil {
		t.Fatal("accepted grant for another side effect")
	}
	save("pack/desk", 1, grant)
	run, err := ws.Start(req)
	if err != nil {
		t.Fatal(err)
	}
	save("tool/update", 1, strings.Replace(def, `"changes"`, `"new_patch"`, 1))
	save("pack/desk", 2, `{"grants":[]}`)
	if _, err := ws.Start(req); err == nil {
		t.Fatal("accepted revoked grant")
	}
	if err := ws.Execute(run.ID); err != nil {
		t.Fatal(err)
	}
	run, err = ws.Get(run.ID)
	if err != nil || run.Status != "in_review" {
		t.Fatal(run.Status, err)
	}
	d := run.Decisions[0]
	proposal := d.Proposal.ToolProposal
	if proposal == nil || proposal.PackRevision != 2 || proposal.Arguments["changes"] == nil {
		t.Fatal(proposal)
	}
	run, err = ws.Decide(run.ID, d.ID, "approved", "", run.Version)
	if err != nil || run.Decisions[0].Status != "approved" || hits.Load() != 0 || len(run.Tools) != 0 {
		t.Fatal(err, hits.Load(), run.Tools)
	}
}
