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
