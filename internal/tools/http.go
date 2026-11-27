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
// Connects any REST system by configuration, not code.
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
	Retries     int
	CacheTTL    time.Duration
	cache       *httpCache
	key         string
}

type httpCache struct {
	mu   sync.Mutex
	data map[string]cacheItem
}

type cacheItem struct {
	value  string
	expiry time.Time
}

// Call executes the tool with the given parameters.
func (t *HTTPTool) Call(ctx context.Context, params map[string]string) (string, error) {
	// Check cache
	cacheKey := t.cacheKey(params)
	if cached, ok := t.cache.get(cacheKey); ok {
		return cached, nil
	}

	url := t.URLTemplate
	for k, v := range params {
		url = strings.ReplaceAll(url, "{"+k+"}", v)
	}

	var lastErr error
	for attempt := 0; attempt <= t.Retries; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		req, err := http.NewRequestWithContext(ctx, t.Method, url, nil)
		if err != nil {
			return "", err
		}
		for k, v := range t.Headers {
			req.Header.Set(k, v)
		}
		if err := t.setAuth(req); err != nil {
			return "", err
		}

		resp, err := t.Client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
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

		body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		if err != nil {
			return "", err
		}
		result := string(body)
		t.cache.set(cacheKey, result, t.CacheTTL)
		return result, nil
	}
	return "", lastErr
}

func (t *HTTPTool) setAuth(req *http.Request) error {
	if t.AuthType == "bearer" && t.KeyFile != "" {
		if t.key == "" {
			b, err := os.ReadFile(t.KeyFile)
			if err != nil {
				return fmt.Errorf("read key: %w", err)
			}
			t.key = strings.TrimSpace(string(b))
		}
		req.Header.Set("Authorization", "Bearer "+t.key)
	}
	return nil
}

func (t *HTTPTool) cacheKey(params map[string]string) string {
	var parts []string
	for k, v := range params {
		parts = append(parts, k+"="+v)
	}
	return t.Name + ":" + strings.Join(parts, "&")
}

func (c *httpCache) get(key string) (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.data[key]
	if !ok || time.Now().After(item.expiry) {
		return "", false
	}
	return item.value, true
}

func (c *httpCache) set(key, value string, ttl time.Duration) {
	if c == nil || ttl == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cacheItem{value: value, expiry: time.Now().Add(ttl)}
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
		Retries:     2,
		cache:       &httpCache{data: map[string]cacheItem{}},
	}
}

// Name returns the tool name.
func (t *HTTPTool) GetName() string { return t.Name }

// Description returns the tool description.
func (t *HTTPTool) GetDescription() string { return t.Description }
