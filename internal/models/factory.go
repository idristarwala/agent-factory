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

	primary, err := primaryGateway(provider)
	if err != nil {
		return nil, err
	}
	judgeProvider := strings.ToLower(os.Getenv("LLM_JUDGE_PROVIDER"))
	if judgeProvider == "" {
		if os.Getenv("LLM_JUDGE_MODEL") != "" {
			return nil, fmt.Errorf("LLM_JUDGE_MODEL requires LLM_JUDGE_PROVIDER")
		}
		return primary, nil
	}
	model := os.Getenv("LLM_JUDGE_MODEL")
	if model == "" {
		return nil, fmt.Errorf("LLM_JUDGE_PROVIDER requires LLM_JUDGE_MODEL")
	}
	var judge Gateway
	switch judgeProvider {
	case "gemini":
		key := os.Getenv("GEMINI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("Gemini judge requires GEMINI_API_KEY")
		}
		judge = NewGemini(key, model, model)
	case "openrouter":
		key := os.Getenv("OPENROUTER_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("OpenRouter judge requires OPENROUTER_API_KEY")
		}
		judge = NewOpenAICompat("https://openrouter.ai/api/v1", key, model, model)
	default:
		return nil, fmt.Errorf("unknown judge provider %q", judgeProvider)
	}
	return roleGateway{primary: primary, judge: judge}, nil
}

type roleGateway struct{ primary, judge Gateway }

func (g roleGateway) Generate(req Request) (*Response, error) {
	if req.CallerRole == "judge" {
		return g.judge.Generate(req)
	}
	return g.primary.Generate(req)
}

func primaryGateway(provider string) (Gateway, error) {
	switch provider {
	case "anthropic":
		return NewAnthropicFromEnv()
	case "gemini":
		return NewGeminiFromEnv()
	case "openrouter":
		if os.Getenv("LLM_BASE_URL") == "" && os.Getenv("OPENROUTER_API_KEY") != "" {
			return NewOpenAICompat("https://openrouter.ai/api/v1", os.Getenv("OPENROUTER_API_KEY"), os.Getenv("LLM_CHEAP_MODEL"), os.Getenv("LLM_FRONTIER_MODEL")), nil
		}
		return NewOpenAICompatFromEnv()
	case "openai", "groq", "openai-compat":
		return NewOpenAICompatFromEnv()
	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER: %s (supported: anthropic, gemini, openai, groq, openrouter)", provider)
	}
}
