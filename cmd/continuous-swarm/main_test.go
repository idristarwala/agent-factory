package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/idristarwala/agent-factory/internal/swarm"
)

func TestConfigureEvidenceChecksSelectedPackBeforeStarting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evidence.yaml")
	config := `tools:
  source: {url: "https://example.org/records/{id}"}
context: {entity: source}
grants:
  - {pack: review, tools: [propose, source], side_effects: [read-only]}
  - {pack: source-only, tools: [source], side_effects: [read-only]}
  - {pack: propose-only, tools: [propose], side_effects: [read-only]}
`
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	for _, pack := range []string{"", "wrong", "source-only", "propose-only"} {
		var r swarm.ContinuousRunner
		if err := configureEvidence(&r, path, pack); err == nil || len(r.Tools) > 0 || len(r.Grants) > 0 {
			t.Fatalf("accepted partial config for %q", pack)
		}
	}
	var r swarm.ContinuousRunner
	if err := configureEvidence(&r, path, "review"); err != nil {
		t.Fatal(err)
	}
	if r.Pack != "review" || r.Recipe.Entity != "source" || r.Tools["source"] == nil || len(r.Grants) == 0 {
		t.Fatal("missing runner configuration")
	}
}
