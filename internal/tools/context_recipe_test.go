package tools

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecipeEvidenceAndFailure(t *testing.T) {
	payload := strings.Repeat("a", 300) + " source fact"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			w.Write([]byte(payload))
		} else {
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	tool := NewHTTPTool("source", "GET", srv.URL+"/ok")
	tool.Client = srv.Client()
	r := ContextRecipe{Entity: "source", Neighbours: "related"}
	grants := []Grant{{Pack: "desk", Tools: []string{"source"}, SideEffects: []string{"read-only"}}}
	if err := r.Authorize("desk", grants); err == nil {
		t.Fatal("missing second grant accepted")
	}
	if _, err := ExecuteRecipe(t.Context(), r, map[string]*HTTPTool{"source": tool}, nil); err == nil {
		t.Fatal("accepted missing tool")
	}
	failed := NewHTTPTool("related", "GET", srv.URL+"/missing")
	failed.Client = srv.Client()
	counts := map[string]int{}
	if _, err := ExecuteRecipeTracked(t.Context(), r, map[string]*HTTPTool{"source": tool, "related": failed}, nil, func(name string) { counts[name]++ }); err == nil {
		t.Fatal("accepted failed tool")
	}
	if counts["source"] != 1 || counts["related"] != 1 {
		t.Fatal(counts)
	}
	r.Neighbours = ""
	if err := r.Authorize("desk", grants); err != nil {
		t.Fatal(err)
	}
	result, err := ExecuteRecipe(t.Context(), r, map[string]*HTTPTool{"source": tool}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ev := r.Evidence(result); len(ev) != 1 || ev[0] != "entity (tool source): "+payload {
		t.Fatalf("evidence %v", ev)
	}
}
