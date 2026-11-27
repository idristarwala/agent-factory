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

func TestResponseMappingInConfiguredEvidence(t *testing.T) {
	body := `{"data":{"request":{"number":9007199254740993,"summary":"Review access","files":[{"url":"image1"}],"optional":null}},"outside":"omit"}`
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, body) }))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "source.yaml")
	config := fmt.Sprintf(`tools:
  lookup:
    url: %s
    response:
      root: $.data.request
      map:
        subject: $.summary
        attachments: $.files[*].url
        id: $.number
        optional: $.optional
context: {entity: lookup}
grants:
  - {pack: desk, tools: [lookup], side_effects: [read-only]}
`, srv.URL)
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadEvidenceConfig(path, "desk")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Tools["lookup"].Client = srv.Client()
	for range 2 {
		result, err := cfg.Gather(t.Context(), "req", nil)
		if err != nil || len(result) != 1 || result[0] != `entity (tool lookup): {"attachments":["image1"],"id":9007199254740993,"optional":null,"subject":"Review access"}` {
			t.Fatal(result, err)
		}
	}
	if calls != 1 {
		t.Fatalf("%d calls, want one cached GET", calls)
	}
}

func TestResponseMappingShapesAndErrors(t *testing.T) {
	for _, tc := range []struct{ root, path, body, want string }{
		{"$.rows[*]", "$[0].v", `{"rows":[{"v":2}]}`, `{"value":2}`},
		{"", "$.rows[*].v", `{"rows":[]}`, `{"value":[]}`},
		{"$.rows", "", "{\"rows\":[1,2]}", "[1,2]"},
		{"", "$.missing", `{"sensitive":"fixture-secret"}`, ""},
		{"$.missing", "", `{"sensitive":"fixture-secret"}`, ""},
		{"$", "", "{} {}", ""},
	} {
		c := responseConfig{Root: tc.root}
		if tc.path != "" {
			c.Map = map[string]string{"value": tc.path}
		}
		m, err := c.compile()
		if err != nil {
			t.Fatal(err)
		}
		got, err := m.apply([]byte(tc.body))
		if got != tc.want || (err != nil) != (tc.want == "") || (err != nil && strings.Contains(err.Error(), "fixture-secret")) {
			t.Fatal(got, err)
		}
	}
	if _, err := (responseConfig{Map: map[string]string{"field": "$["}}).compile(); err == nil {
		t.Fatal("accepted invalid selector")
	}
}
