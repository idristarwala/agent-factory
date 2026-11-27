package tools

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAuthorizedSourceSample(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("cursor") != "a&b" || r.Header.Get("Authorization") != "Bearer fixture" {
			t.Error("incorrect source request")
		}
		w.Write([]byte(`{"records":[{"id":9007199254740993}]}`))
	}))
	defer srv.Close()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "key"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "sources.yaml")
	if err := os.WriteFile(path, []byte("tools:\n  meter:\n    url: "+srv.URL+"\n    query: [cursor]\n    auth: {type: bearer, key_file: key}\ngrants:\n  - {pack: meter-review, tools: [meter], side_effects: [read-only]}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSourceSampler(path)
	if err != nil {
		t.Fatal(err)
	}
	s.config.Tools["meter"].Client = srv.Client()
	if _, err := s.Sample(t.Context(), "other", "meter", nil); !errors.Is(err, ErrSourceDenied) {
		t.Fatal(err)
	}
	if _, err := s.Sample(t.Context(), "meter-review", "meter", map[string]string{"other": "bad"}); !errors.Is(err, ErrSourceParameter) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("invalid sampling request reached source")
	}
	for i := 0; i < 2; i++ {
		raw, err := s.Sample(t.Context(), "meter-review", "meter", map[string]string{"cursor": "a&b"})
		if err != nil || string(raw) != `{"records":[{"id":9007199254740993}]}` {
			t.Fatal(string(raw), err)
		}
	}
	if calls != 2 {
		t.Fatal("source sample must be fresh")
	}
}
