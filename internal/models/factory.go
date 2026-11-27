package models

import (
	"fmt"
	"os"
	"strings"
)

// NewGatewayFromEnv returns the appropriate Gateway based on LLM_PROVIDER.
// Supported: "gemini" (native), "openai" / "groq" / "openrouter" (OpenAI-compatible).
// Defaults to gemini if GEMINI_API_KEY is set, otherwise openai-compat.

func NewGatewayFromEnv() (Gateway, error) {
	provider := strings.ToLower(os.Getenv("LLM_PROVIDER"))
	if provider == "" {
		if os.Getenv("GEMINI_API_KEY") != "" || os.Getenv("LLM_API_KEY") != "" {
			// Prefer native Gemini when a Gemini-style key is present
			if _, err := NewGeminiFromEnv(); err == nil {
				provider = "gemini"
			}
		}
		if provider == "" {
			provider = "openai"
		}
	}

	switch provider {
	case "anthropic":
		return NewAnthropicFromEnv()
	case "gemini":
		return NewGeminiFromEnv()
	case "openai", "groq", "openrouter", "openai-compat":
		return NewOpenAICompatFromEnv()
	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER: %s (supported: anthropic, gemini, openai, groq, openrouter)", provider)
	}
}
