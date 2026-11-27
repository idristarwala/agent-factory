package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadinessChecksConfiguredToolGrantsAndCredentials(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	key := filepath.Join(dir, "token")
	t.Setenv("FACTORY_STORE_MODE", "memory")
	t.Setenv("FACTORY_EVIDENCE_CONFIG", path)
	t.Setenv("FACTORY_PACK", "desk")
	t.Setenv("FACTORY_SOURCE_CONFIG", path)
	body := "tools: {lookup: {url: 'https://example.org/items', auth: {type: bearer, key_file: token}}}\ncontext: {entity: lookup}\ngrants: [{pack: desk, tools: [lookup, propose], side_effects: [read-only]}]"
	write := func(path, data string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(path, body)
	for _, value := range []string{"", "   ", "example-credential"} {
		if value != "" {
			write(key, value)
		}
		err := runReadiness()
		if (err == nil) != (value == "example-credential") {
			t.Fatalf("unexpected readiness result: %v", err)
		}
	}
	write(path, strings.Replace(body, "lookup, propose", "lookup", 1))
	if err := runReadiness(); err == nil {
		t.Fatal("missing proposal grant passed")
	}
	t.Setenv("FACTORY_PACK", "")
	t.Setenv("FACTORY_EVIDENCE_CONFIG", "")
	write(path, strings.Replace(body, "items'", "items/{id}'", 1))
	if err := runReadiness(); err == nil {
		t.Fatal("invalid sampler passed")
	}
	t.Setenv("FACTORY_SOURCE_CONFIG", "")
	if err := runReadiness(); err != nil {
		t.Fatal(err)
	}
}
