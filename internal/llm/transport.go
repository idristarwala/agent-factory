package llm

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"time"
)

// NewHTTPClient builds an HTTP client with optional custom CA and proxy control.
// Supports the TLS and proxy requirements from issue #11.
func NewHTTPClient(caFile, proxy string) (*http.Client, error) {
	transport := &http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
	}

	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM(caCert)
		transport.TLSClientConfig = &tls.Config{RootCAs: caPool}
	}

	if proxy == "none" {
		transport.Proxy = nil
	}
	// else uses environment proxy by default

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}, nil
}
