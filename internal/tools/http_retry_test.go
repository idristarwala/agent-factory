package tools

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestConfiguredSearchReplay(t *testing.T) {
	for _, auth := range []string{"header", "basic"} {
		t.Run(auth, func(t *testing.T) {
			var calls atomic.Int32
			value := "quote \" slash \\ newline\n{other}"
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
					t.Error("incorrect search request")
				}
				if auth == "header" && r.Header.Get("X-Search-Key") != "fixture" {
					t.Error("header credential missing")
				}
				if auth == "basic" {
					u, p, ok := r.BasicAuth()
					if !ok || u != "reader" || p != "fixture" {
						t.Error("basic authentication missing")
					}
				}
				var body struct{ Term string }
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Term != value {
					t.Errorf("search term mismatch: %v", err)
				}
				if calls.Add(1) == 1 {
					w.WriteHeader(429)
					return
				}
				io.WriteString(w, "related")
			}))
			defer srv.Close()
			dir := t.TempDir()
			write := func(name, stringValue string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, name), []byte(stringValue), 0600); err != nil {
					t.Fatal(err)
				}
			}
			write("key", "fixture")
			write("ca.pem", string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})))
			authConfig := "type: header, name: X-Search-Key"
			if auth == "basic" {
				authConfig = "type: basic, username: reader"
			}
			write("tools.yaml", fmt.Sprintf(`tools:
  lookup:
    method: POST
    url: %s
    side_effect: read-only
    body: {term: "{label}"}
    auth: {%s, key_file: key}
    tls: {ca_file: ca.pem}
    proxy: none
    limits: {retries: 1, timeout: 2s, cache_ttl: 0s}
context: {neighbours: lookup}
grants:
  - {pack: team, tools: [lookup], side_effects: [read-only]}
`, srv.URL, authConfig))
			c, err := LoadEvidenceConfig(filepath.Join(dir, "tools.yaml"), "team")
			if err != nil {
				t.Fatal(err)
			}
			tool := c.registry["lookup"]
			tr := tool.Client.Transport.(*http.Transport)
			if tr.Proxy != nil || tool.Client.Timeout != 2*time.Second || tool.CacheTTL != 0 {
				t.Fatal("configured transport limits not applied")
			}
			if _, err := srv.Certificate().Verify(x509.VerifyOptions{Roots: tr.TLSClientConfig.RootCAs}); err != nil {
				t.Fatal(err)
			}
			// Only the loopback dialer is substituted. TLS uses the configured roots.
			tr.DialContext = srv.Client().Transport.(*http.Transport).DialContext
			result, err := c.Gather(t.Context(), "a", map[string]any{"label": value})
			if err != nil || len(result) != 1 || !strings.Contains(result[0], "related") || calls.Load() != 2 {
				t.Fatalf("evidence %v, calls %d, err %v", result, calls.Load(), err)
			}
		})
	}
}

func TestEvidenceCancellation(t *testing.T) {
	var calls atomic.Int32
	received := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(received)
		}
		w.WriteHeader(503)
	}))
	defer srv.Close()
	tool := NewHTTPTool("lookup", "GET", srv.URL)
	tool.Client = srv.Client()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := tool.Call(ctx, nil); done <- err }()
	<-received
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatalf("calls %d, error %v", calls.Load(), err)
	}
}

func TestHTTPToolCacheSizeAndTimeout(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/timeout":
			<-r.Context().Done()
		case "/large":
			io.WriteString(w, strings.Repeat("x", 512*1024+1))
		default:
			io.WriteString(w, "record")
		}
	}))
	defer srv.Close()
	tool := NewHTTPTool("source", "GET", srv.URL+"/ok")
	tool.Client = srv.Client()
	tool.Retries = 0
	for i := 0; i < 2; i++ {
		if v, err := tool.Call(t.Context(), nil); err != nil || v != "record" {
			t.Fatalf("unexpected result %q %v", v, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("cache missed")
	}
	tool.URLTemplate = srv.URL + "/large"
	if _, err := tool.Call(t.Context(), nil); err == nil {
		t.Fatal("oversized response accepted")
	}
	tool.URLTemplate = srv.URL + "/timeout?token=private-fixture-value"
	tool.Client.Timeout = 50 * time.Millisecond
	if _, err := tool.Call(t.Context(), nil); err == nil || strings.Contains(err.Error(), "private-fixture-value") {
		t.Fatal("missing or unredacted timeout error")
	}
}
