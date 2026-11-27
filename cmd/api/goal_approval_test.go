package main

import (
	"github.com/idristarwala/agent-factory/internal/artifact"
	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/revision"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCitedGoalApproval(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	s := revision.NewMemory()
	ref, err := artifact.Save(s, artifact.Ref{}, "notes.md", []byte("Review sample values"))
	if err != nil {
		t.Fatal(err)
	}
	refs := []artifact.Ref{ref}
	evidence, err := artifact.Read(s, refs)
	if err != nil {
		t.Fatal(err)
	}
	g := &proposalGateway{t: t, result: `{"goal":"Review","record_type":"sample","fields":[{"name":"value","type":"string"}],"criteria":[{"text":"Inspect value","sources":[2]}]}`}
	ws := factory.Workspace{Store: s, Gateway: g}
	if _, err = createChatFactory(&ws, "Review", refs, evidence, "", 0); err == nil {
		t.Fatal("accepted nonexistent source")
	}
	g.result = strings.Replace(g.result, "[2]", "[1]", 1)
	draft, err := createChatFactory(&ws, "Review", refs, evidence, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	id := draft["proposal"].(string)
	h, err := revision.Documents(s, "factory/"+id)
	if err != nil || len(h) != 0 {
		t.Fatal("factory created before approval")
	}
	g.result = strings.Replace(g.result, "Inspect value", "Inspect all values", 1)
	if _, err = createChatFactory(&ws, "Revise", refs, evidence, id, 1); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	goalApprovalRoute(mux, s)
	for _, tc := range []struct {
		body string
		want int
	}{{`{"version":1}`, 409}, {`{"version":2}`, 200}, {`{"version":2}`, 200}, {`{"version":3}`, 409}} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/chat/proposals/"+id+"/approve", strings.NewReader(tc.body)))
		if w.Code != tc.want {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
	h, err = revision.Documents(s, "factory/"+id)
	if err != nil || len(h) != 1 || h[0].CreatedBy != "human" || !strings.Contains(string(h[0].Data), `"proposal_version":2`) {
		t.Fatal(h, err)
	}
}
