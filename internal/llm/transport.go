package llm

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
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
		caPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("gateway CA file contains no valid certificates")
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: caPool}
	}

	switch proxy {
	case "", "env":
		transport.Proxy = http.ProxyFromEnvironment
	case "none": // Direct connections, including when the process has proxy settings.
	default:
		return nil, fmt.Errorf("unknown proxy mode %q", proxy)
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}, nil
}
