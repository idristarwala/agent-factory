package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPTool is a config-driven HTTP tool for issue #12.
// Connects any REST system by configuration, not code.
type HTTPTool struct {
	Name        string
	Description string
	Method      string
	URLTemplate string
	AuthType    string
	KeyFile     string
	Headers     map[string]string
	SideEffect  string // read-only, reversible, irreversible
	Client      *http.Client
}

// Call executes the tool with the given parameters.
// Path templating is basic ({id} replacement).
func (t *HTTPTool) Call(ctx context.Context, params map[string]string) (string, error) {
	url := t.URLTemplate
	for k, v := range params {
		url = strings.ReplaceAll(url, "{"+k+"}", v)
	}

	req, err := http.NewRequestWithContext(ctx, t.Method, url, nil)
	if err != nil {
		return "", err
	}
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}
	if t.AuthType == "bearer" && t.KeyFile != "" {
		// Key loading would go here; redacted in logs.
		req.Header.Set("Authorization", "Bearer [REDACTED]")
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// NewHTTPTool creates a tool from a simple definition.
func NewHTTPTool(name, method, urlTemplate string) *HTTPTool {
	return &HTTPTool{
		Name:        name,
		Method:      method,
		URLTemplate: urlTemplate,
		SideEffect:  "read-only",
		Client:      &http.Client{Timeout: 20 * time.Second},
		Headers:     map[string]string{},
	}
}
