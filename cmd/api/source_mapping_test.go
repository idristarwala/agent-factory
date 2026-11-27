package main

import (
	"encoding/json"
	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/revision"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMappingPreviewRevisionAndRun(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	store := revision.NewMemory()
	mux := http.NewServeMux()
	sourceMappingRoutes(mux, store)
	post := func(path, body string, status int) string {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader(body)))
		if w.Code != status {
			t.Fatalf("status %d: %s", w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	const sample = `{"items":[{"key":9007199254740993,"tenant":"north","info":{"reading":9007199254740995}},{"key":7,"tenant":"north","info":{"reading":42}}]}`
	const spec = `{"collection":"/items","record_type":"meter","identity":["/tenant","/key"],"fields":{"reading":"/info/reading"}}`
	request := `{"mapping":` + spec + `,"sample":` + sample + `,"expected":0}`
	bad := strings.Replace(request, "9007199254740993", "7", 1)
	if got := post("/api/sources/preview", bad, 200); !strings.Contains(got, `"valid":false`) || strings.Contains(got, `"record":`) {
		t.Fatal(got)
	}
	post("/api/sources/mappings/meters", bad, 422)
	post("/api/sources/mappings/meters", request, 200)
	post("/api/sources/mappings/meters", request, 409)
	changed := strings.Replace(strings.Replace(request, `"expected":0`, `"expected":1`, 1), `"/info/reading"`, `"/key"`, 1)
	post("/api/sources/mappings/meters", changed, 200)
	if _, err := revision.SaveDocument(store, "factory/meters", 0, "human", []byte(`{"record_type":"meter","fields":{"reading":"integer"}}`)); err != nil {
		t.Fatal(err)
	}
	ws := factory.Workspace{Store: store}
	r, err := ws.Start(factory.RunRequest{Description: "review", Factory: "meters", Source: &factory.SourceInput{SourceRef: factory.SourceRef{Mapping: "meters", Revision: 1}, Payload: json.RawMessage(sample)}})
	if err != nil {
		t.Fatal(err)
	}
	r, err = ws.Get(r.ID)
	if err != nil || r.Source.Revision != 1 || r.Records[0].ID != `["north",9007199254740993]` || r.Records[0].Attributes["reading"] != json.Number("9007199254740995") {
		t.Fatal(r, err)
	}
}

func TestApprovedSourceDrift(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	store := revision.NewMemory()
	mux := http.NewServeMux()
	sourceMappingRoutes(mux, store)
	const spec = `{"collection":"","record_type":"meter","identity":["/id"],"fields":{"reading":"/reading","note":"/note"}}`
	const approved = `[{"id":1,"reading":2,"note":null},{"id":2,"reading":5}]`
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/sources/mappings/meters", strings.NewReader(`{"mapping":`+spec+`,"sample":`+approved+`,"expected":0}`)))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if _, err := revision.SaveDocument(store, "factory/f", 0, "human", []byte(`{"record_type":"meter","fields":{"reading":"number"}}`)); err != nil {
		t.Fatal(err)
	}
	ws := factory.Workspace{Store: store}
	req := factory.RunRequest{Description: "check", Factory: "f", Source: &factory.SourceInput{SourceRef: factory.SourceRef{Mapping: "meters", Revision: 1}, Payload: json.RawMessage(`[{"id":3,"reading":9,"extra":true},{"id":"4","reading":5},{"id":5},{"id":6,"reading":"broken"},{"id":7,"reading":1},{"id":7,"reading":2}]`)}}
	if _, err := ws.Start(req); err == nil || !strings.Contains(err.Error(), "/id") {
		t.Fatal(err)
	}
	if runs, err := ws.List(); err != nil || len(runs) != 0 {
		t.Fatal(runs, err)
	}
	req.Source.AllowPartial = true
	r, err := ws.Start(req)
	if err != nil {
		t.Fatal(err)
	}
	r, err = ws.Get(r.ID)
	if err != nil || len(r.Records) != 1 || r.Records[0].ID != "3" || len(r.ImportProblems) != 5 || r.ImportProblems[0].Index != 1 || !strings.Contains(r.ImportProblems[1].Reason, "missing") {
		t.Fatal(r, err)
	}
	req.Source.Payload = json.RawMessage(`[{"id":8}]`)
	if _, err := ws.Start(req); err == nil {
		t.Fatal("queued empty import")
	}
	// Documents approved before the shape metadata existed retain their original mapping semantics.
	if _, err := revision.SaveDocument(store, "source-mapping/legacy", 0, "human", []byte(spec)); err != nil {
		t.Fatal(err)
	}
	req.Source.Mapping = "legacy"
	if _, err := ws.Start(req); err != nil {
		t.Fatal(err)
	}
}
