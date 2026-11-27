package tools

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfiguredRecordEvidence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/records/a%2Fb" || r.URL.Query().Get("label") != "x & y" || r.Header.Get("Authorization") != "Bearer fixture" {
			t.Error("incorrect record request")
		}
		fmt.Fprint(w, "source record")
	}))
	defer srv.Close()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "key"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf(`tools:
  source:
    url: %s/records/{id}
    query: [label]
    auth: {type: bearer, key_file: key}
context: {entity: source}
grants:
  - {pack: claims, tools: [source], side_effects: [read-only]}
`, srv.URL)
	path := filepath.Join(dir, "evidence.yml")
	for _, tc := range []struct {
		pack, config string
		valid        bool
	}{
		{"wrong", config, false}, {"claims", strings.Replace(config, "entity: source", "entity: missing", 1), false},
		{"claims", config + "unexpected: yes\n", false}, {"claims", config, true},
	} {
		if err := os.WriteFile(path, []byte(tc.config), 0600); err != nil {
			t.Fatal(err)
		}
		c, err := LoadEvidenceConfig(path, tc.pack)
		if !tc.valid {
			if err == nil {
				t.Fatal("accepted invalid config")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		c.Tools["source"].Client = srv.Client()
		evidence, err := c.Gather(t.Context(), "a/b", map[string]any{"id": "wrong", "label": "x & y"})
		if err != nil || len(evidence) != 1 || evidence[0] != "entity (tool source): source record" {
			t.Fatalf("unexpected evidence %v, %v", evidence, err)
		}
	}
}
