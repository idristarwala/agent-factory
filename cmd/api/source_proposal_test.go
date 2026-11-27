package main

import (
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type proposalGateway struct {
	calls  int
	result string
	t      *testing.T
}

func (g *proposalGateway) Generate(req models.Request) (*models.Response, error) {
	g.calls++
	if strings.Contains(req.User, "PRIVATE-VALUE") || req.CallerRole != "steward" {
		g.t.Fatal("incorrect steward context")
	}
	return &models.Response{Content: g.result}, nil
}

func TestStewardMappingChoicesAndApproval(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	s := revision.NewMemory()
	g := &proposalGateway{t: t, result: `{"fields":[{"name":"label","path":"/name"}]}`}
	mux := http.NewServeMux()
	sourceProposalRoutes(mux, s, g)
	sourceMappingRoutes(mux, s)
	req := func(path, body string, want int) string {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader(body)))
		if w.Code != want {
			t.Fatalf("status %d: %s", w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	sample := `{"data":[{"id":1,"name":"PRIVATE-VALUE"},{"id":2,"name":"other"}]}`
	body := `{"sample":` + sample + `,"goal":"review labels","record_type":"claim","expected":0`
	path := "/api/sources/proposals/import"
	if !strings.Contains(req(path, body+"}", 200), "choose_collection") {
		t.Fatal("missing collection choice")
	}
	body += `,"collection":"/data"`
	if !strings.Contains(req(path, body+"}", 200), "choose_identity") || g.calls != 0 {
		t.Fatal("ambiguous identity was resolved automatically")
	}
	body += `,"identity":["/id"]}`
	if !strings.Contains(req(path, body, 200), `"created_by":"steward"`) {
		t.Fatal("missing steward draft")
	}
	req(path, body, 409)
	g.result = `{"fields":[{"name":"wrong","path":"/invented"}]}`
	req(path, strings.Replace(body, `"expected":0`, `"expected":1`, 1), 502)
	h, _ := revision.Documents(s, "source-proposal/import")
	if len(h) != 1 || strings.Contains(string(h[0].Data), "PRIVATE-VALUE") {
		t.Fatal("invalid draft history")
	}
	if !strings.Contains(req("/api/sources/mappings/import", `{"expected":0,"proposal_revision":1,"sample":`+sample+`}`, 200), `"proposal_revision":1`) {
		t.Fatal("missing approved proposal reference")
	}
}
