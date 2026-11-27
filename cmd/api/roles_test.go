package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/revision"
)

func TestRoleBoundaries(t *testing.T) {
	t.Setenv("API_TOKEN", "a")
	t.Setenv("FACTORY_OPERATOR_TOKEN", "o")
	t.Setenv("FACTORY_VIEWER_TOKEN", "v")
	mux := http.NewServeMux()
	store := revision.NewMemory()
	ws := &factory.Workspace{Store: store}
	configRoutes(mux, store)
	memoryRoutes(mux, store)
	chatRoute(mux, ws, nil)
	runRoutes(mux, ws, nil)
	mux.HandleFunc("/api/extension", apiRoute(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(201) }, "POST"))
	call := func(method, path, body, token string, want int) {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, want, w.Body.String())
		}
	}
	const memory = "/api/memory/tickets"
	const edit = `{"path":"notes","content":"draft","expected":0}`
	call("POST", memory, edit, "v", 403)
	call("GET", memory+"?path=notes", "", "v", 404)
	call("POST", memory, edit, "o", 200)
	call("GET", memory+"?path=notes", "", "v", 200)
	const def = "/api/definitions/tool/example"
	const create = `{"data":{"name":"example"},"expected":0}`
	call("POST", def, create, "o", 403)
	call("POST", def, create, "a", 200)
	call("POST", "/api/chat", `{"message":"Create a factory to triage tickets"}`, "o", 403)
	call("POST", "/api/chat", `{"message":"status"}`, "o", 200)
	call("DELETE", "/api/goals/missing", "", "v", 403)
	call("DELETE", "/api/goals/missing", "", "o", 404)
	call("POST", "/api/extension", "", "o", 403)
	call("POST", "/api/extension", "", "a", 201)
	t.Setenv("API_TOKEN", "")
	t.Setenv("FACTORY_OPERATOR_TOKEN", "")
	call("GET", def, "", "", 401)
	call("GET", def, "", "v", 200)
	call("OPTIONS", def, "", "", 204)
	t.Setenv("API_TOKEN", "v")
	call("POST", def, create, "v", 503)
	t.Setenv("API_TOKEN", "")
	t.Setenv("FACTORY_VIEWER_TOKEN", "")
	call("POST", "/api/extension", "", "", 201)
}
