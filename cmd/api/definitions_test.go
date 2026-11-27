package main

import (
	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefinitionRoutesRejectStaleEdits(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	s := revision.NewMemory()
	m := http.NewServeMux()
	definitionRoutes(m, rules.NewStoredEngine(s), agents.NewStoredRegistry(s))
	cases := []struct {
		path, body string
		status     int
	}{
		{"/api/agents", `{"Name":"steward","Role":"steward","Expected":0}`, 200},
		{"/api/agents", `{"Name":"steward","Role":"worker","Expected":0}`, 409},
		{"/api/rules", `{"ID":"source","Description":"check source","Expected":0}`, 200},
		{"/api/rules", `{"ID":"source","Expected":1,"Promote":1}`, 200},
		{"/api/rules", `{"ID":"source","Expected":1,"Description":"stale"}`, 409},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest("POST", c.path, strings.NewReader(c.body)))
		if w.Code != c.status {
			t.Fatalf("%s got %d: %s", c.path, w.Code, w.Body.String())
		}
	}
}
