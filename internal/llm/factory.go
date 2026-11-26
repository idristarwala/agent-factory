package llm

import (
	"fmt"
	"os"
	"strings"

	"github.com/idristarwala/agent-factory/internal/secrets"
)

// NewFromEnv creates a Router wrapping the configured adapter.
// Secrets come from SECRETS_DIR (default /etc/secrets) with env fallback.
func NewFromEnv() (*Router, error) {
	sec := secrets.New()

	provider := strings.ToLower(os.Getenv("LLM_PROVIDER"))
	if provider == "" {
		if _, err := sec.Get("gemini-api-key"); err == nil {
			provider = "gemini"
		} else if _, err := sec.Get("llm-api-key"); err == nil {
			provider = "openai"
		} else {
			return nil, fmt.Errorf("no LLM provider configured")
		}
	}

	key, _ := sec.Get("gemini-api-key")
	if key == "" {
		key, _ = sec.Get("llm-api-key")
	}

	cfg := map[string]string{
		"api_key":  key,
		"base_url": os.Getenv("LLM_BASE_URL"),
	}

	c, err := New(provider, cfg)
	if err != nil {
		return nil, err
	}
	return NewRouter(c), nil
}
