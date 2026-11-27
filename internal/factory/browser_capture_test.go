package factory

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/schema"
)

type captureCheckingGateway struct {
	expected, field string
	image           string
	worker, judge   int
	change          func()
	live            models.Gateway
}

func (g *captureCheckingGateway) Generate(req models.Request) (*models.Response, error) {
	if !strings.Contains(req.User, g.expected) || len(req.Images) != 1 || req.Images[0].Mime != "image/jpeg" {
		return nil, errors.New("missing captured evidence")
	}
	response := fmt.Sprintf(`{"%s":"Support"}`, g.field)
	if req.CallerRole == "worker" {
		g.worker++
		g.image = req.Images[0].Base64
		g.change()
	} else {
		g.judge++
		if req.Images[0].Base64 != g.image {
			return nil, errors.New("judge received a different image")
		}
		response = "APPROVE: evidence matches"
	}
	if g.live != nil {
		return g.live.Generate(req)
	}
	return &models.Response{Content: response, Model: "fixture"}, nil
}

func exerciseBrowserRun(t *testing.T, live models.Gateway) {
	t.Helper()
	for _, schemaName := range []string{"ticket", "asset"} {
		t.Run(schemaName, func(t *testing.T) {
			var changed atomic.Bool
			page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				if changed.Load() {
					fmt.Fprint(w, "<main>Replacement page</main>")
					return
				}
				fmt.Fprint(w, `<main></main><script>document.querySelector("main").textContent="Verified destination: Support"</script>`)
			}))
			defer page.Close()
			field := "team"
			if schemaName == "asset" {
				field = "maintainer"
			}
			gw := &captureCheckingGateway{expected: "Verified destination: Support", field: field, change: func() { changed.Store(true) }, live: live}
			ws := Workspace{Store: revision.NewMemory(), Gateway: gw}
			if _, err := agents.NewStoredRegistry(ws.Store).Save(agents.AgentVersion{Name: "judge", Role: "judge", ModelTier: "frontier", CreatedBy: "human", Config: map[string]string{"max_output_tokens": "2200"}}, 0); err != nil {
				t.Fatal(err)
			}
			save := func(key string, expected int, body string) {
				t.Helper()
				if _, err := revision.SaveDocument(ws.Store, key, expected, "human", []byte(body)); err != nil {
					t.Fatal(err)
				}
			}
			tool := fmt.Sprintf(`{"kind":"playwright","side_effect":"read-only","origins":[%q],"url_field":"reference","screenshot":true}`, page.URL)
			save("tool/reader", 0, tool)
			save("factory/example", 0, fmt.Sprintf(`{"record_type":%q,"fields":{%q:"string"},"browser_tool":"reader"}`, schemaName, field))
			save("pack/example", 0, `{"grants":[]}`)
			req := RunRequest{Description: "Set " + field + " to the exact verified destination from the rendered page.", Factory: "example", Records: []schema.Entity{{ID: "case", Type: schemaName, Attributes: map[string]any{field: "unknown", "reference": page.URL}}}}
			if _, err := ws.Start(req); err == nil {
				t.Fatal("missing grant accepted")
			}
			save("pack/example", 1, `{"grants":[{"pack":"example","tools":["reader"],"side_effects":["read-only"]}]}`)
			run, err := ws.Start(req)
			if err != nil {
				t.Fatal(err)
			}
			save("tool/reader", 1, strings.Replace(tool, page.URL, "https://replacement.invalid", 1))
			if err = ws.Execute(run.ID); err != nil {
				t.Fatal(err)
			}
			run, err = ws.Get(run.ID)
			if err != nil || run.Status != "in_review" || len(run.Decisions) != 1 || gw.worker != 1 || gw.judge != 1 || run.Tools["reader"] != 1 {
				t.Fatal(run.Status, run.Error, err, gw.worker, gw.judge)
			}
			d := run.Decisions[0]
			c, err := ReadBrowserCapture(ws.Store, d.BrowserCapture)
			if err != nil || c.ToolRevision != 1 || c.PackRevision != 2 || c.Source != page.URL || c.CapturedAt.IsZero() || !strings.Contains(c.Text, gw.expected) || len(c.Image) == 0 || !strings.Contains(strings.Join(d.Proposal.Evidence, " "), c.ID) {
				t.Fatal("incorrect capture", err, c.ID)
			}
			run, err = ws.Decide(run.ID, d.ID, "approved", "", run.Version)
			if err != nil {
				t.Fatal(err)
			}
			if run.Records[0].Attributes[field] != "Support" {
				t.Fatal(run.Records[0])
			}
			if live == nil {
				replay, err := ws.ReviewedReplay(fmt.Sprintf("%s@%d", run.ID, run.Version), graph.NewMemoryGraph(), nil)
				if err != nil {
					t.Fatal(err)
				}
				report, err := replay(rules.VersionedRule{ID: "test", Description: "Use the verified destination"}, nil)
				if err != nil || !report.Cases[0].BaselinePassed || !report.Cases[0].CandidatePassed {
					t.Fatal(report, err)
				}
			}
			later, err := ws.Start(req)
			if err != nil {
				t.Fatal(err)
			}
			if err := ws.Execute(later.ID); err != nil {
				t.Fatal(err)
			}
			later, err = ws.Get(later.ID)
			if err != nil || later.Status != "failed" || len(later.Calls) != 0 || later.Tools["reader"] != 1 {
				t.Fatal(later.Status, err)
			}
			if live != nil {
				for _, c := range run.Calls {
					t.Logf("%s %s %d tokens", c.Role, c.Model, c.Total)
				}
			}
		})
	}
}

func TestPinnedBrowserCaptureReviewAndReplay(t *testing.T) {
	script := filepath.Join(t.TempDir(), "capture.mjs")
	err := os.WriteFile(script, []byte(`let data="";for await(const c of process.stdin)data+=c;const req=JSON.parse(data);const body=await(await fetch(req.url)).text();process.stdout.write(JSON.stringify({url:req.url,text:body.match(/Verified destination: Support/)?.[0]??"changed",image:"/9j/4A=="}))`), 0600)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FACTORY_PLAYWRIGHT_SCRIPT", script)
	exerciseBrowserRun(t, nil)
}
