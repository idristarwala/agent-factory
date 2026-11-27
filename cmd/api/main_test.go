package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProtectedCrossOriginRoute(t *testing.T) {
	t.Setenv("API_TOKEN", "example")
	t.Setenv("CORS_ORIGIN", "https://ui.example")
	h := apiRoute(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) }, http.MethodPost)
	for _, tc := range []struct {
		name, method, origin, token string
		want                        int
		allowOrigin                 string
	}{
		{"preflight", "OPTIONS", "https://ui.example", "", 204, "https://ui.example"},
		{"other origin", "OPTIONS", "https://other.example", "", 204, ""},
		{"missing token", "POST", "https://ui.example", "", 401, "https://ui.example"},
		{"valid token", "POST", "https://ui.example", "Bearer example", 201, "https://ui.example"},
		{"wrong method", "GET", "https://ui.example", "Bearer example", 405, "https://ui.example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/chat", nil)
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Authorization", tc.token)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tc.want || w.Header().Get("Access-Control-Allow-Origin") != tc.allowOrigin {
				t.Fatalf("status %d, CORS origin %q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}
