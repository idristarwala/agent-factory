package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHTTPToolPathTemplating(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/items/123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":"123","title":"test"}`))
	}))
	defer server.Close()

	tool := NewHTTPTool("get_item", "GET", server.URL+"/items/{id}")
	tool.Client = server.Client()
	result, err := tool.Call(context.Background(), map[string]string{"id": "123"})
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Error("expected response body")
	}
}

func TestHTTPToolBearerAuth(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.txt")
	os.WriteFile(keyPath, []byte("secret-token"), 0600)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer secret-token" {
			t.Errorf("expected bearer token, got %q", auth)
		}
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	tool := NewHTTPTool("auth_test", "GET", server.URL+"/test")
	tool.Client = server.Client()
	tool.AuthType = "bearer"
	tool.KeyFile = keyPath
	_, err := tool.Call(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
}
