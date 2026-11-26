package llm

import (
	"fmt"
	"os"
	"strings"
)

// NewFromEnv creates a Router wrapping the configured adapter.
// Honors LLM_PROVIDER, GEMINI_API_KEY, LLM_API_KEY, LLM_BASE_URL.
func NewFromEnv() (*Router, error) {
	provider := strings.ToLower(os.Getenv("LLM_PROVIDER"))
	if provider == "" {
		if os.Getenv("GEMINI_API_KEY") != "" {
			provider = "gemini"
		} else if os.Getenv("LLM_API_KEY") != "" {
			provider = "openai"
		} else {
			return nil, fmt.Errorf("no LLM provider configured")
		}
	}

	cfg := map[string]string{
		"api_key":  firstNonEmpty(os.Getenv("GEMINI_API_KEY"), os.Getenv("LLM_API_KEY")),
		"base_url": os.Getenv("LLM_BASE_URL"),
	}

	c, err := New(provider, cfg)
	if err != nil {
		return nil, err
	}
	return NewRouter(c), nil
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}
