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

func TestGatewayReadinessIsOfflineAndOptIn(t *testing.T) {
	t.Setenv("FACTORY_STORE_MODE", "memory")
	dir := t.TempDir()
	cfg := filepath.Join(dir, "router.yaml")
	t.Setenv("LLM_GATEWAY_CONFIG", cfg)
	write := func(path, value string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(cfg, "gateways: {local: {base_url: 'http://127.0.0.1:1', auth: {type: bearer, key_file: token}}}\nmodels: {cheap: {gateway: local, protocol: openai, model: sample}}")
	if err := runReadiness(); err != nil {
		t.Fatal(err)
	}
	if err := runReadiness("--gateway"); err == nil {
		t.Fatal("missing file passed")
	}
	write(filepath.Join(dir, "token"), " \n")
	if err := runReadiness("--gateway"); err == nil {
		t.Fatal("empty file passed")
	}
	write(filepath.Join(dir, "token"), "fixture")
	if err := runReadiness("--gateway"); err != nil {
		t.Fatal(err)
	}
	write(cfg, "sensitive-config-marker: 1")
	if err := runReadiness("--gateway"); err == nil || strings.Contains(err.Error(), "sensitive-config-marker") {
		t.Fatal(err)
	}
	t.Setenv("LLM_GATEWAY_CONFIG", "")
	t.Setenv("LLM_PROVIDER", "openrouter")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_JUDGE_PROVIDER", "")
	t.Setenv("LLM_JUDGE_MODEL", "")
	t.Setenv("OPENROUTER_API_KEY", " ")
	if err := runReadiness("--gateway"); err == nil {
		t.Fatal("empty environment key passed")
	}
	t.Setenv("OPENROUTER_API_KEY", "fixture")
	if err := runReadiness("--gateway"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLM_BASE_URL", "relative-endpoint")
	t.Setenv("LLM_API_KEY", "fixture")
	if err := runReadiness("--gateway"); err == nil {
		t.Fatal("malformed endpoint passed")
	}
}
