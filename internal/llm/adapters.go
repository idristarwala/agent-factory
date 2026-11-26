package llm

import (
	"context"
	"os"

	"github.com/idristarwala/agent-factory/internal/models"
)

func init() {
	Register("gemini", newGeminiAdapter)
	Register("openai", newOpenAIAdapter)
	Register("groq", newOpenAIAdapter)
	Register("openrouter", newOpenAIAdapter)
}

func newGeminiAdapter(cfg map[string]string) (Client, error) {
	// Fall back to env for now
	if cfg["api_key"] != "" {
		os.Setenv("GEMINI_API_KEY", cfg["api_key"])
	}
	gw, err := models.NewGeminiFromEnv()
	if err != nil {
		return nil, err
	}
	return &modelsAdapter{gw: gw}, nil
}

func newOpenAIAdapter(cfg map[string]string) (Client, error) {
	if cfg["base_url"] != "" {
		os.Setenv("LLM_BASE_URL", cfg["base_url"])
	}
	if cfg["api_key"] != "" {
		os.Setenv("LLM_API_KEY", cfg["api_key"])
	}
	gw, err := models.NewOpenAICompatFromEnv()
	if err != nil {
		return nil, err
	}
	return &modelsAdapter{gw: gw}, nil
}

// modelsAdapter wraps the existing models.Gateway behind the new Client interface.
type modelsAdapter struct {
	gw models.Gateway
}

func (a *modelsAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Convert to old Request
	var system, user string
	for _, m := range req.Messages {
		if m.Role == "system" {
			system = m.Content
		} else if m.Role == "user" {
			user = m.Content
		}
	}
	if user == "" && system != "" {
		user = system
		system = ""
	}

	temp := 0.0
	if req.Temperature != nil {
		temp = *req.Temperature
	}

	oldReq := models.Request{
		Tier:        models.Tier(req.Tier),
		System:      system,
		User:        user,
		Temperature: temp,
		MaxTokens:   req.MaxTokens,
	}

	resp, err := a.gw.Generate(oldReq)
	if err != nil {
		return nil, &Error{Type: "TRANSPORT", Message: err.Error(), Retryable: true, Cause: err}
	}

	return &ChatResponse{
		Content:      resp.Content,
		Model:        resp.Model,
		LogicalModel: req.Model,
		Usage: Usage{
			Input:  resp.Usage["prompt_tokens"],
			Output: resp.Usage["completion_tokens"],
			Total:  resp.Usage["total_tokens"],
		},
	}, nil
}

func (a *modelsAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}
