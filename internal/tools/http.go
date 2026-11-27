package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
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
	QueryParams  []string // param names that go to query string
	AuthType     string
	KeyFile      string
	AuthHeader   string
	Username     string
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

var bodyParameter = regexp.MustCompile(`\{[a-zA-Z0-9_.-]+\}`)

// Call executes the tool with path/body/query templating, key from file, retries, and cache.
func (t *HTTPTool) Call(ctx context.Context, params map[string]string) (string, error) {
	if t.SideEffect != "read-only" && t.SideEffect != "" {
		return "", fmt.Errorf("non-read-only tools must become proposals")
	}

	urlStr := t.URLTemplate
	for k, v := range params {
		urlStr = strings.ReplaceAll(urlStr, "{"+k+"}", url.PathEscape(v))
	}

	if err := validateDestination(t.URLTemplate, urlStr); err != nil {
		return "", err
	}

	// Add query params
	if len(t.QueryParams) > 0 {
		u, err := url.Parse(urlStr)
		if err == nil {
			q := u.Query()
			for _, name := range t.QueryParams {
				if v, ok := params[name]; ok {
					q.Set(name, v)
				}
			}
			u.RawQuery = q.Encode()
			urlStr = u.String()
		}
	}

	body := t.BodyTemplate
	if body != "" {
		jsonBody := json.Valid([]byte(body))
		body = bodyParameter.ReplaceAllStringFunc(body, func(token string) string {
			value, ok := params[token[1:len(token)-1]]
			if !ok {
				return token
			}
			if !jsonBody {
				return value
			}
			encoded, _ := json.Marshal(value)
			return string(encoded[1 : len(encoded)-1])
		})
	}
	cacheable := t.CacheTTL > 0 && body == "" && t.Method == http.MethodGet
	if cacheable {
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
		if attempt > 0 {
			delay := time.NewTimer(time.Duration(min(attempt, 10)) * 200 * time.Millisecond)
			select {
			case <-ctx.Done():
				delay.Stop()
				return "", ctx.Err()
			case <-delay.C:
			}
		}
		req, err := http.NewRequestWithContext(ctx, t.Method, urlStr, strings.NewReader(body))
		if err != nil {
			return "", err
		}
		for k, v := range t.Headers {
			req.Header.Set(k, v)
		}
		if t.AuthType != "" {
			key, err := os.ReadFile(t.KeyFile)
			if err != nil {
				return "", fmt.Errorf("read key file: %w", err)
			}
			value := strings.TrimSpace(string(key))
			switch t.AuthType {
			case "bearer":
				req.Header.Set("Authorization", "Bearer "+value)
			case "header":
				req.Header.Set(t.AuthHeader, value)
			case "basic":
				req.SetBasicAuth(t.Username, value)
			default:
				return "", fmt.Errorf("unsupported HTTP tool authentication")
			}
		}

		resp, err := t.Client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			lastErr = fmt.Errorf("HTTP tool request failed")
			continue
		}

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("http %d", resp.StatusCode)
			resp.Body.Close()
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("http %d", resp.StatusCode)
		}

		respBody, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024+1))
		if err != nil {
			return "", err
		}
		if len(respBody) > 512*1024 {
			return "", fmt.Errorf("tool response exceeds 512 KiB")
		}
		result := string(respBody)

		if cacheable {
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
		Client:      publicClient(),
		Headers:     map[string]string{},
		Retries:     2,
		CacheTTL:    10 * time.Minute,
	}
}
