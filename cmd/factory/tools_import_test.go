package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/tools"
)

const importSpec = `openapi: 3.0.3
info: {title: Tickets, version: "1"}
servers: [{url: https://service.example/api}]
security: [{token: []}]
components:
  securitySchemes:
    token: {type: apiKey, in: header, name: X-API-Key}
  parameters:
    ticket: {in: path, name: id, required: true, schema: {type: string}}
paths:
  /tickets/{id}:
    parameters: [{$ref: '#/components/parameters/ticket'}]
    get:
      operationId: lookup
      parameters: [{in: query, name: team, required: true, schema: {type: string}}]
      responses: {"200": {description: OK}}
    delete:
      operationId: remove
      responses: {"204": {description: Removed}}
`

func TestImportReviewedEvidenceTool(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "openapi.yaml")
	write := func(path, data string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(input, importSpec)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.EscapedPath() != "/api/tickets/a%2Fb" || r.URL.Query().Get("team") != "level two" || r.Header.Get("X-API-Key") != "test-token" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		io.WriteString(w, `{"owner":"support"}`)
	}))
	defer srv.Close()
	args := []string{"import", "-server", srv.URL + "/api", "-operation", "lookup", "-key-file", "credential", input}
	var out bytes.Buffer
	if err := importTools(args, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "tools.yaml")
	write(path, out.String()+"\ncontext: {entity: lookup}\n")
	if _, err := tools.LoadEvidenceConfig(path, "desk"); err == nil {
		t.Fatal("import granted itself access")
	}
	write(path, out.String()+"\ncontext: {entity: lookup}\ngrants: [{pack: desk, tools: [lookup], side_effects: [read-only]}]\n")
	write(filepath.Join(dir, "credential"), "test-token")
	cfg, err := tools.LoadEvidenceConfig(path, "desk")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Tools["lookup"].Client = srv.Client()
	if _, err := cfg.Gather(t.Context(), "a/b", nil); err == nil || calls != 0 {
		t.Fatal("missing required query reached service")
	}
	result, err := cfg.Gather(t.Context(), "a/b", map[string]any{"team": "level two"})
	if err != nil || calls != 1 || !strings.Contains(strings.Join(result, " "), "support") {
		t.Fatal(result, err)
	}
	for _, arg := range [][]string{{"import", input}, {"import", "-operation", "remove", input}, {"import", "-operation", "unknown", input}} {
		out.Reset()
		if err := importTools(arg, &out, io.Discard); err == nil || out.Len() != 0 {
			t.Fatalf("expected atomic failure for %v", arg)
		}
	}
}

func TestImportEnvironmentCredential(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "openapi.yaml")
	if err := os.WriteFile(spec, []byte(importSpec), 0600); err != nil {
		t.Fatal(err)
	}
	const name = "FACTORY_IMPORT_TEST_KEY"
	t.Setenv(name, "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "runtime-only" {
			t.Error("missing imported credential")
		}
		io.WriteString(w, "record")
	}))
	defer srv.Close()
	var out bytes.Buffer
	if err := importTools([]string{"import", "-operation", "lookup", "-server", srv.URL, "-key-env", name, spec}, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "tools.yaml")
	if err := os.WriteFile(path, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tools.ValidateEvidenceFile(path); err != nil {
		t.Fatal(err)
	}
	out.WriteString("\ncontext: {entity: lookup}\ngrants: [{pack: desk, tools: [lookup], side_effects: [read-only]}]\n")
	if err := os.WriteFile(path, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := tools.LoadEvidenceConfig(path, "desk")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(name, "runtime-only")
	cfg.Tools["lookup"].Client = srv.Client()
	if _, err := cfg.Gather(t.Context(), "one", map[string]any{"team": "support"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"-operation", "lookup", "-key-file", "token", "-key-env", name},
		{"-operation", "lookup", "-key-env", "bad-name"},
		{"-key-env", name},
	} {
		out.Reset()
		if err := importTools(append(append([]string{"import"}, args...), spec), &out, io.Discard); err == nil || out.Len() != 0 {
			t.Fatalf("expected invalid credential selection to fail without a draft: %v", args)
		}
	}
}
