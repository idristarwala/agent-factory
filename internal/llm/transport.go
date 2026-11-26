package llm

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

// NewHTTPClient builds an HTTP client with optional custom CA and proxy.
func NewHTTPClient(tlsCfg TLSConfig, proxy string) (*http.Client, error) {
	transport := &http.Transport{}

	if tlsCfg.CAFile != "" {
		caCert, err := os.ReadFile(tlsCfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA file")
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: pool}
	}

	if proxy == "none" {
		transport.Proxy = nil
	}
	// "env" uses the default proxy from environment

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}, nil
}
