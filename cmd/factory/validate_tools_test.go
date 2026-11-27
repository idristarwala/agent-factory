package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/idristarwala/agent-factory/internal/tools"
)

func TestValidateImportedDraftBeforeDeployment(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "api.yaml")
	if err := os.WriteFile(spec, []byte(importSpec), 0600); err != nil {
		t.Fatal(err)
	}
	var draft bytes.Buffer
	if err := importTools([]string{"import", "-operation", "lookup", "-key-file", "missing-token", spec}, &draft, io.Discard); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FACTORY_HTTP_ALLOWED_HOSTS", "")
	path := filepath.Join(dir, "tools.yaml")
	check := func(data string, valid bool) {
		t.Helper()
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if err := validateFiles([]string{"--tools", path}); (err == nil) != valid {
			t.Fatalf("valid=%t: %v", valid, err)
		}
	}
	check(draft.String(), true)
	check(`{"tools":{"list":{"url":"https://tools.example/mcp","mcp_tool":"list","side_effect":"read-only","tls":{"ca_file":"missing-ca"}}}}`, true)
	if _, err := tools.LoadEvidenceConfig(path, "desk"); err == nil {
		t.Fatal("deployed config unexpectedly accepted")
	}
	for _, body := range []string{
		"tools: {}",
		"tools: {one: {url: 'https://tool.example', method: POST}}",
		"tools: {one: {url: 'https://tool.example', unexpected: true}}",
		"tools: {one: {url: 'https://tool.example', limits: {timeout: -1s}}}",
		"tools: {one: {url: 'https://tool.example', response: {root: '['}}}",
		draft.String() + "\ncontext: {entity: absent}\n",
		draft.String() + "\n---\n" + draft.String(),
	} {
		check(body, false)
	}
}
