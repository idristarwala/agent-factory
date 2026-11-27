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

func load(path string) (*Router, error) {
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
		u, err := url.Parse(g.BaseURL)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
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
		client.Transport = authTransport{client.Transport, g.Auth.Type, relative(dir, g.Auth.KeyFile)}
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
		headers := llm.ApplyExtraHeaders(g.Headers, m.Headers)
		base := strings.TrimRight(g.BaseURL, "/")
		var adapter llm.Client
		if m.Envelope != nil && m.Protocol != "openai-envelope" {
			return nil, fmt.Errorf("envelope provided for %q protocol", m.Protocol)
		}
		switch m.Protocol {
		case "openai", "openai-envelope":
			a := llm.NewOpenAIAdapter(base, "", headers)
			if m.Protocol == "openai-envelope" {
				if err := m.Envelope.Validate(); err != nil {
					return nil, fmt.Errorf("model %q: %w", name, err)
				}
				a.Envelope = m.Envelope
			}
			a.Client = clients[m.Gateway]
			adapter = a
		case "anthropic":
			a := llm.NewAnthropicAdapter(strings.TrimSuffix(base, "/v1"), "", headers)
			a.Client = clients[m.Gateway]
			adapter = a
		case "gemini":
			a := llm.NewGeminiAdapter(base, "", headers)
			a.Client = clients[m.Gateway]
			adapter = a
		default:
			return nil, fmt.Errorf("unknown protocol %q", m.Protocol)
		}
		if m.TokenParam != "" && ((m.Protocol != "openai" && m.Protocol != "openai-envelope") || (m.TokenParam != "max_tokens" && m.TokenParam != "max_completion_tokens")) {
			return nil, fmt.Errorf("invalid token_param for model %q", name)
		}
		if m.ReasoningBudget != nil && (m.Protocol != "gemini" || *m.ReasoningBudget < 0) {
			return nil, fmt.Errorf("invalid reasoning_budget for model %q", name)
		}
		router.routes[name] = route{m, retryClient{inner: &llm.RecordingClient{Inner: adapter, Logger: logger}}}
	}
	return router, nil
}
