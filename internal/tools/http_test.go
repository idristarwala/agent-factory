package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	result, err := tool.Call(context.Background(), map[string]string{"id": "123"})
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Error("expected response body")
	}
}
