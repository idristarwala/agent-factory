package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// HTTPTool is a config-driven HTTP tool for issue #12.
type HTTPTool struct {
	Name        string
	Description string
	Method      string
	URLTemplate string
	AuthType    string
	KeyFile     string
	Headers     map[string]string
	SideEffect  string
	Client      *http.Client
	cache       map[string]string
	cacheMu     sync.RWMutex
}

// Call executes the tool with path templating, retries, and basic cache.
func (t *HTTPTool) Call(ctx context.Context, params map[string]string) (string, error) {
	url := t.URLTemplate
	for k, v := range params {
		url = strings.ReplaceAll(url, "{"+k+"}", v)
	}

	// Simple cache check
	t.cacheMu.RLock()
	if cached, ok := t.cache[url]; ok {
		t.cacheMu.RUnlock()
		return cached, nil
	}
	t.cacheMu.RUnlock()

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, t.Method, url, nil)
		if err != nil {
			return "", err
		}
		for k, v := range t.Headers {
			req.Header.Set(k, v)
		}
		if t.AuthType == "bearer" && t.KeyFile != "" {
			key, err := os.ReadFile(t.KeyFile)
			if err != nil {
				return "", fmt.Errorf("read key file: %w", err)
			}
			req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(key)))
		}

		resp, err := t.Client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			lastErr = fmt.Errorf("http %d", resp.StatusCode)
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("http %d", resp.StatusCode)
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		if err != nil {
			return "", err
		}
		result := string(body)

		// Cache success
		t.cacheMu.Lock()
		if t.cache == nil {
			t.cache = make(map[string]string)
		}
		t.cache[url] = result
		t.cacheMu.Unlock()

		return result, nil
	}
	return "", lastErr
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
		cache:       make(map[string]string),
	}
}
