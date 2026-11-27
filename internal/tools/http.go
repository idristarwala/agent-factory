package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// HTTPTool is a config-driven HTTP tool for issue #12.
type HTTPTool struct {
	Name         string
	Description  string
	Method       string
	URLTemplate  string
	BodyTemplate string
	AuthType     string
	KeyFile      string
	Headers      map[string]string
	SideEffect   string
	Client       *http.Client
	Retries      int
	CacheTTL     time.Duration
	cache        map[string]cacheEntry
	mu           sync.Mutex
}

type cacheEntry struct {
	body string
	exp  time.Time
}

// Call executes the tool with path/body templating, key from file, retries, and cache.
func (t *HTTPTool) Call(ctx context.Context, params map[string]string) (string, error) {
	if t.SideEffect != "read-only" && t.SideEffect != "" {
		return "", fmt.Errorf("non-read-only tools must become proposals")
	}

	urlStr := t.URLTemplate
	for k, v := range params {
		urlStr = strings.ReplaceAll(urlStr, "{"+k+"}", url.PathEscape(v))
	}

	var body io.Reader
	if t.BodyTemplate != "" {
		b := t.BodyTemplate
		for k, v := range params {
			b = strings.ReplaceAll(b, "{"+k+"}", v)
		}
		body = bytes.NewReader([]byte(b))
	}

	if t.CacheTTL > 0 && body == nil {
		t.mu.Lock()
		if e, ok := t.cache[urlStr]; ok && time.Now().Before(e.exp) {
			t.mu.Unlock()
			return e.body, nil
		}
		t.mu.Unlock()
	}

	var lastErr error
	attempts := t.Retries + 1
	if attempts < 1 {
		attempts = 1
	}
	for attempt := 0; attempt < attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, t.Method, urlStr, body)
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
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("http %d", resp.StatusCode)
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("http %d", resp.StatusCode)
		}

		respBody, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		if err != nil {
			return "", err
		}
		result := string(respBody)

		if t.CacheTTL > 0 && body == nil {
			t.mu.Lock()
			if t.cache == nil {
				t.cache = make(map[string]cacheEntry)
			}
			t.cache[urlStr] = cacheEntry{body: result, exp: time.Now().Add(t.CacheTTL)}
			t.mu.Unlock()
		}
		return result, nil
	}
	return "", lastErr
}

func NewHTTPTool(name, method, urlTemplate string) *HTTPTool {
	return &HTTPTool{
		Name:        name,
		Method:      method,
		URLTemplate: urlTemplate,
		SideEffect:  "read-only",
		Client:      &http.Client{Timeout: 20 * time.Second},
		Headers:     map[string]string{},
		Retries:     2,
		CacheTTL:    10 * time.Minute,
	}
}
