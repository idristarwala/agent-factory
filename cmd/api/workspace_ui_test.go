package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPackagedWorkspaceRoutes(t *testing.T) {
	dir := t.TempDir()
	mux := http.NewServeMux()
	if err := workspaceUI(mux, dir); err == nil {
		t.Fatal("accepted missing UI")
	}
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"index.html": "<title>Workspace</title>", "assets/app.js": "window.loaded=true"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := workspaceUI(mux, dir); err != nil {
		t.Fatal(err)
	}
	for path, code := range map[string]int{"/": 200, "/assets/app.js": 200, "/assets/": 404, "/api/unknown": 404, "/assets/..%2findex.html": 404} {
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, httptest.NewRequest("GET", path, nil))
		if res.Code != code {
			t.Errorf("%s: got %d, want %d", path, res.Code, code)
		}
	}
}
