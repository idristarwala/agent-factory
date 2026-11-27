package tools

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestConfiguredMCPEvidence(t *testing.T) {
	for _, asJSON := range []bool{true, false} {
		t.Run(fmt.Sprintf("JSONResponse=%t", asJSON), func(t *testing.T) {
			peer := mcp.NewServer(&mcp.Implementation{Name: "fixture", Version: "1"}, nil)
			type args struct {
				ID string `json:"id"`
			}
			type row struct {
				Subject string `json:"subject"`
				Hidden  string `json:"internal"`
			}
			mcp.AddTool(peer, &mcp.Tool{Name: "read_request"}, func(ctx context.Context, r *mcp.CallToolRequest, a args) (*mcp.CallToolResult, row, error) {
				if a.ID == "fail" {
					return nil, row{}, errors.New("sensitive fixture detail")
				}
				return nil, row{Subject: "Request " + a.ID, Hidden: "exclude this"}, nil
			})
			handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return peer }, &mcp.StreamableHTTPOptions{JSONResponse: asJSON})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer fixture-key" {
					t.Error("missing credential")
					w.WriteHeader(401)
					return
				}
				handler.ServeHTTP(w, r)
			}))
			defer srv.Close()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "key"), []byte("fixture-key"), 0600); err != nil {
				t.Fatal(err)
			}
			config := fmt.Sprintf(`tools:
  lookup:
    url: %s
    mcp_tool: read_request
    side_effect: read-only
    auth: {type: bearer, key_file: key}
    body: {id: "{id}"}
    response:
      map: {title: $.subject}
context: {entity: lookup}
grants:
  - {pack: desk, tools: [lookup], side_effects: [read-only]}
`, srv.URL)
			path := filepath.Join(dir, "mcp.yml")
			if err := os.WriteFile(path, []byte(config), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadEvidenceConfig(path, "ungranted"); err == nil {
				t.Fatal("accepted pack without grant")
			}
			cfg, err := LoadEvidenceConfig(path, "desk")
			if err != nil {
				t.Fatal(err)
			}
			cfg.Tools["lookup"].Client = srv.Client()
			result, err := cfg.Gather(t.Context(), "r19", nil)
			if err != nil || len(result) != 1 || result[0] != `entity (tool lookup): {"title":"Request r19"}` {
				t.Fatal(result, err)
			}
			_, err = cfg.Gather(t.Context(), "fail", nil)
			if err == nil || strings.Contains(err.Error(), "sensitive") || strings.Contains(err.Error(), "fixture-key") {
				t.Fatal(err)
			}
		})
	}
}
