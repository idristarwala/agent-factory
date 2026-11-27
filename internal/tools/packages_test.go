package tools

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinnedPackageInFactoryManifest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"count":9007199254740993}`)
	}))
	defer srv.Close()
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(dir, "factory.yaml")
	pkg := "tools: {lookup: {url: " + srv.URL + "}}\n"
	pin := func(content string) string {
		return fmt.Sprintf("  - {path: tools.yaml, sha256: '%x'}\n", sha256.Sum256([]byte(content)))
	}
	manifest := "tool_packages:\n" + pin(pkg) + "evidence:\n  grants: [{pack: review, tools: [lookup], side_effects: [read-only]}]\n"
	write("tools.yaml", pkg)
	write("factory.yaml", manifest)
	if err := ValidateEvidenceFile(file); err != nil {
		t.Fatal(err)
	}
	sampler, err := LoadSourceSampler(file)
	if err != nil {
		t.Fatal(err)
	}
	sampler.config.Tools["lookup"].Client = srv.Client()
	if _, err := sampler.Sample(t.Context(), "other", "lookup", nil); !errors.Is(err, ErrSourceDenied) {
		t.Fatal(err)
	}
	got, err := sampler.Sample(t.Context(), "review", "lookup", nil)
	if err != nil || string(got) != `{"count":9007199254740993}` {
		t.Fatal(string(got), err)
	}
	for _, tc := range []struct{ name, content, manifest string }{
		{"changed", pkg + "\n", manifest},
		{"self-grant", pkg + "grants: []\n", "tool_packages:\n" + pin(pkg+"grants: []\n")},
		{"collision", pkg, "tool_packages:\n" + pin(pkg) + pin(pkg)},
		{"extra document", pkg + "---\ncontext: {}\n", "tool_packages:\n" + pin(pkg+"---\ncontext: {}\n")},
		{"path escape", pkg, strings.Replace(manifest, "path: tools.yaml", "path: ../tools.yaml", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			write("tools.yaml", tc.content)
			write("factory.yaml", tc.manifest)
			if ValidateEvidenceFile(file) == nil {
				t.Error("invalid package passed validation")
			}
			if _, err := LoadSourceSampler(file); err == nil {
				t.Error("invalid package loaded at runtime")
			}
		})
	}
}
