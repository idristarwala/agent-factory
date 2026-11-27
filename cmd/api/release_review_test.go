package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/revision"
)

func TestPackReviewKeepsApprovedEvidence(t *testing.T) {
	t.Setenv("API_TOKEN", "admin-test")
	t.Setenv("FACTORY_VIEWER_TOKEN", "viewer-test")
	st := revision.NewMemory()
	mux := http.NewServeMux()
	configRoutes(mux, st)
	call := func(method, path, token, body string, status int) revision.Document {
		t.Helper()
		req := httptest.NewRequest(method, "/api/releases/support/"+path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s: got %d %s", path, w.Code, w.Body.String())
		}
		var doc revision.Document
		if status == 200 {
			if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
		}
		return doc
	}
	propose := func(expected int, commit string) {
		t.Helper()
		_, err := revision.SaveDocument(st, "pack-release/support", expected, "publisher", []byte(`{"status":"proposed","commit":"`+commit+`","pack_bytes":"YQ==","evidence_bytes":"Yg=="}`))
		if err != nil {
			t.Fatal(err)
		}
	}
	first := `{"expected":1,"decision":"approve","note":"reviewed"}`
	propose(0, "first")
	call("GET", "active", "viewer-test", "", 404)
	call("POST", "review", "viewer-test", first, 403)
	review := call("POST", "review", "admin-test", first, 200)
	if review.Version != 2 || review.CreatedBy != "human" {
		t.Fatal(review)
	}
	if got := call("POST", "review", "admin-test", first, 200); got.Version != 2 {
		t.Fatal(got)
	}
	propose(2, "next")
	call("POST", "review", "admin-test", first, 409)
	call("POST", "review", "admin-test", `{"expected":3,"decision":"reject"}`, 200)
	active := call("GET", "active", "viewer-test", "", 200)
	if string(active.Data) != string(review.Data) || active.Version != 2 {
		t.Fatal(active)
	}
	history, err := revision.Documents(st, "pack-release/support")
	if err != nil || len(history) != 4 || !strings.Contains(string(history[0].Data), `"status":"proposed"`) {
		t.Fatal(history, err)
	}
}
