package main

import (
	"encoding/json"
	"github.com/idristarwala/agent-factory/internal/mapping"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProfileSourceEnvelope(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/sources/profile", strings.NewReader(`{"data":{"rows":[{"key":9007199254740992,"owner":{"a/b":"sensitive-value"}},{"key":9007199254740993,"owner":null}]},"other":[{"key":"a"},{"key":"a"}]}`))
	w := httptest.NewRecorder()
	profileSource(w, r)
	var resp struct{ Collections []mapping.CollectionProfile }
	if w.Code != 200 || strings.Contains(w.Body.String(), "sensitive-value") || json.Unmarshal(w.Body.Bytes(), &resp) != nil || len(resp.Collections) != 3 {
		t.Fatal("unexpected sample response", w.Code)
	}
	rows := resp.Collections[1]
	if rows.Path != "/data/rows" || rows.Fields[0].Path != "/key" || !rows.Fields[0].UniqueInSample || rows.Fields[2].Path != "/owner/a~1b" || rows.Fields[2].Missing != 1 || rows.Fields[1].Types["null"] != 1 || resp.Collections[2].Fields[0].UniqueInSample {
		t.Fatalf("unexpected field profiles %+v", resp)
	}
}
