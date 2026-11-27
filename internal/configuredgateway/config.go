// Package configuredgateway selects the deployed gateway from a model routing file.
package configuredgateway

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/idristarwala/agent-factory/internal/llm"
	"github.com/idristarwala/agent-factory/internal/obs"
	"gopkg.in/yaml.v3"
)

type gatewayConfig struct {
	BaseURL string `yaml:"base_url"`
	Auth    struct {
		Type    string
		KeyFile string `yaml:"key_file"`
	} `yaml:"auth"`
	Headers map[string]string
	TLS     struct {
		CAFile string `yaml:"ca_file"`
	}
	Proxy string
}
type modelConfig struct {
	Retry                    retryOverrides      `yaml:"retry"`
	SchemaMode               string              `yaml:"schema_mode"`
	Envelope                 *llm.OpenAIEnvelope `yaml:"envelope"`
	Gateway, Protocol, Model string
	Headers                  map[string]string
	ReasoningBudget          *int   `yaml:"reasoning_budget"`
	TokenParam               string `yaml:"token_param"`
}

type authTransport struct {
	next            http.RoundTripper
	scheme, keyFile string
}

func (a authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Del("Authorization")
	r.Header.Del("x-api-key")
	r.Header.Del("x-goog-api-key")
	if a.scheme != "none" {
		key, err := llm.LoadKey(a.keyFile, "")
		if err != nil {
			return nil, err
		}
		if key == "" {
			return nil, errors.New("empty gateway credential")
		}
		switch a.scheme {
		case "bearer":
			r.Header.Set("Authorization", "Bearer "+key)
		case "google_api_key":
			r.Header.Set("x-goog-api-key", key)
		case "anthropic_api_key":
			r.Header.Set("x-api-key", key)
		}
	}
	return a.next.RoundTrip(r)
}

func relative(dir, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dir, path)
}

func validGatewayURL(base string) bool {
	u, err := url.Parse(base)
	return err == nil && u.Hostname() != "" && (u.Scheme == "https" || u.Scheme == "http") && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == ""
}

func load(path string, checkKeys bool) (*Router, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var config struct {
		Gateways map[string]gatewayConfig
		Models   map[string]modelConfig
	}
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err = decoder.Decode(&config); err != nil {
		return nil, err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("expected one gateway configuration document")
	}
	clients := map[string]*http.Client{}
	dir := filepath.Dir(path)
	for name, g := range config.Gateways {
		if !validGatewayURL(g.BaseURL) {
			return nil, fmt.Errorf("invalid base URL for gateway %q", name)
		}
		switch g.Auth.Type {
		case "bearer", "google_api_key", "anthropic_api_key":
			if g.Auth.KeyFile == "" {
				return nil, fmt.Errorf("gateway %q needs a key_file", name)
			}
		case "none":
		default:
			return nil, fmt.Errorf("unknown auth type %q", g.Auth.Type)
		}
		client, err := llm.NewHTTPClient(relative(dir, g.TLS.CAFile), g.Proxy)
		if err != nil {
			return nil, err
		}
		keyPath := relative(dir, g.Auth.KeyFile)
		if checkKeys && g.Auth.Type != "none" {
			key, err := llm.LoadKey(keyPath, "")
			if err != nil || key == "" {
				return nil, errors.New("gateway key file is unreadable or empty")
			}
		}
		client.Transport = authTransport{client.Transport, g.Auth.Type, keyPath}
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("gateway endpoint redirected") }
		clients[name] = client
	}
	if len(config.Models) == 0 {
		return nil, errors.New("configure at least one model")
	}
	router := &Router{routes: map[string]route{}}
	logger := obs.New()
	for name, m := range config.Models {
		g, ok := config.Gateways[m.Gateway]
		if !ok || strings.TrimSpace(m.Model) == "" || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("invalid route for model %q", name)
		}
		entry, err := buildRoute(m, g.BaseURL, "", llm.ApplyExtraHeaders(g.Headers, m.Headers), clients[m.Gateway], logger)
		if err != nil {
			return nil, fmt.Errorf("model %q: %w", name, err)
		}
		router.routes[name] = entry
	}
	return router, nil
}

func buildRoute(m modelConfig, baseURL, key string, headers map[string]string, client *http.Client, logger *obs.Logger) (route, error) {
	base := strings.TrimRight(baseURL, "/")
	if m.SchemaMode != "" && !(m.Protocol == "anthropic" && m.SchemaMode == "forced_tool") && !((m.Protocol == "openai" || m.Protocol == "openai-envelope") && m.SchemaMode == "non_strict") {
		return route{}, errors.New("invalid schema_mode for protocol")
	}
	var adapter llm.Client
	if m.Envelope != nil && m.Protocol != "openai-envelope" {
		return route{}, fmt.Errorf("envelope provided for %q protocol", m.Protocol)
	}
	switch m.Protocol {
	case "openai", "openai-envelope":
		a := llm.NewOpenAIAdapter(base, key, headers)
		if m.Protocol == "openai-envelope" {
			if err := m.Envelope.Validate(); err != nil {
				return route{}, err
			}
			a.Envelope = m.Envelope
		}
		a.Client = client
		adapter = a
	case "anthropic":
		a := llm.NewAnthropicAdapter(strings.TrimSuffix(base, "/v1"), key, headers)
		a.Client = client
		adapter = a
	case "gemini":
		a := llm.NewGeminiAdapter(base, key, headers)
		a.Client = client
		adapter = a
	default:
		return route{}, fmt.Errorf("unknown protocol %q", m.Protocol)
	}
	if m.TokenParam != "" && ((m.Protocol != "openai" && m.Protocol != "openai-envelope") || (m.TokenParam != "max_tokens" && m.TokenParam != "max_completion_tokens")) {
		return route{}, errors.New("invalid token_param")
	}
	if m.ReasoningBudget != nil && (m.Protocol != "gemini" || *m.ReasoningBudget < 0) {
		return route{}, errors.New("invalid reasoning_budget")
	}
	policy, err := m.Retry.resolve()
	if err != nil {
		return route{}, err
	}
	return route{m, retryClient{policy: policy, inner: &llm.RecordingClient{Inner: adapter, Logger: logger}}}, nil
}
