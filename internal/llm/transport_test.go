package llm

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewHTTPClientNoCA(t *testing.T) {
	c, err := NewHTTPClient("", "none")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Error("expected client")
	}
}

func TestGatewayTransportTrustAndProxy(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	defer server.Close()
	ca := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	client, err := NewHTTPClient(ca, "none")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 || client.Transport.(*http.Transport).Proxy != nil {
		t.Fatal("expected direct TLS request")
	}
	for _, mode := range []string{"env", ""} {
		client, err := NewHTTPClient("", mode)
		if err != nil || client.Transport.(*http.Transport).Proxy == nil {
			t.Fatalf("environment proxy unavailable for mode %q", mode)
		}
	}
	if _, err := NewHTTPClient("", "enb"); err == nil {
		t.Fatal("accepted invalid proxy mode")
	}
	if err := os.WriteFile(ca, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewHTTPClient(ca, "none"); err == nil {
		t.Fatal("accepted invalid CA")
	}
}
