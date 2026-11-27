package llm

import (
	"context"

	"github.com/idristarwala/agent-factory/internal/models"
)

func init() {
	Register("gemini", newGeminiAdapter)
	Register("openai", newOpenAIAdapter)
	Register("groq", newOpenAIAdapter)
	Register("openrouter", newOpenAIAdapter)
}

func newGeminiAdapter(cfg map[string]string) (Client, error) {
	apiKey := cfg["api_key"]
	if apiKey == "" {
		return nil, &Error{Type: "AUTH", Message: "api_key required for gemini adapter", Retryable: false}
	}
	cheap := cfg["cheap_model"]
	frontier := cfg["frontier_model"]
	gw := models.NewGemini(apiKey, cheap, frontier)
	return &modelsAdapter{gw: gw}, nil
}

func newOpenAIAdapter(cfg map[string]string) (Client, error) {
	baseURL := cfg["base_url"]
	apiKey := cfg["api_key"]
	if baseURL == "" || apiKey == "" {
		return nil, &Error{Type: "AUTH", Message: "base_url and api_key required for openai-compat adapter", Retryable: false}
	}
	cheap := cfg["cheap_model"]
	frontier := cfg["frontier_model"]
	gw := models.NewOpenAICompat(baseURL, apiKey, cheap, frontier)
	return &modelsAdapter{gw: gw}, nil
}

// modelsAdapter wraps the existing models.Gateway behind the new Client interface.
type modelsAdapter struct {
	gw models.Gateway
}

func (a *modelsAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	schema, err := compileOutputSchema(req.ResponseSchema)
	if err != nil {
		return nil, err
	}

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
		Schema:      req.ResponseSchema,
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

	if err := validateOutput(schema, resp.Content); err != nil {
		return nil, err
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
