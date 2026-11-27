package configuredgateway

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strconv"
	"strings"

	"github.com/idristarwala/agent-factory/internal/llm"
	"github.com/idristarwala/agent-factory/internal/models"
)

type route struct {
	model  modelConfig
	client llm.Client
}
type Router struct{ routes map[string]route }

func NewFromEnv() (models.Gateway, error) {
	if path := os.Getenv("LLM_GATEWAY_CONFIG"); path != "" {
		return load(path)
	}
	return models.NewGatewayFromEnv()
}

// CheckJudgeRoutes validates the complete panel before any model is called.
func (r *Router) CheckJudgeRoutes(names []string) error {
	seen := map[string]bool{}
	for _, name := range names {
		entry, ok := r.routes[name]
		if !ok {
			return fmt.Errorf("unknown judge route %q", name)
		}
		model := strings.ToLower(strings.TrimSpace(entry.model.Model))
		if model == "" || seen[model] {
			return fmt.Errorf("judge routes must select distinct model IDs")
		}
		seen[model] = true
	}
	return nil
}

func (r *Router) logicalName(req llm.ChatRequest) string {
	logical := req.Model
	if logical == "" {
		logical = string(req.Tier)
		if logical == "" {
			logical = "cheap"
		}
		if _, ok := r.routes["judge"]; ok && req.CallerRole == llm.RoleJudge {
			logical = "judge"
		}
	}
	return logical
}

func (r *Router) resolve(req llm.ChatRequest) (llm.Client, llm.ChatRequest, error) {
	logical := r.logicalName(req)
	entry, ok := r.routes[logical]
	if !ok {
		return nil, req, &llm.Error{Type: "BAD_REQUEST", Message: fmt.Sprintf("unknown logical model %q", logical)}
	}
	if req.CallerRole == llm.RoleWorker && (logical == "frontier" || logical == "judge" || logical == "council" || req.Tier == llm.TierFrontier || req.Tier == llm.TierCouncil) {
		return nil, req, &llm.Error{Type: "CAPABILITY", Message: "worker cannot request a privileged model"}
	}
	req.Model = entry.model.Model
	req.Metadata = maps.Clone(req.Metadata)
	if req.Metadata == nil {
		req.Metadata = map[string]string{}
	}
	if entry.model.ReasoningBudget != nil {
		req.Metadata["reasoning_budget"] = strconv.Itoa(*entry.model.ReasoningBudget)
	}
	if entry.model.TokenParam != "" {
		req.Metadata["token_param"] = entry.model.TokenParam
	}
	return entry.client, req, nil
}

func (r *Router) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	client, wire, err := r.resolve(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.Chat(ctx, wire)
	if err == nil {
		resp.LogicalModel = r.logicalName(req)
		if resp.Model == "" {
			resp.Model = wire.Model
		}
	}
	return resp, err
}

func (r *Router) Stream(ctx context.Context, req llm.ChatRequest) (llm.Stream, error) {
	client, wire, err := r.resolve(req)
	if err != nil {
		return nil, err
	}
	return client.Stream(ctx, wire)
}

func (r *Router) Generate(req models.Request) (*models.Response, error) {
	if req.JudgeRoute != "" && req.CallerRole != "judge" {
		return nil, &llm.Error{Type: "CAPABILITY", Message: "only judges can select a judge route"}
	}
	var messages []llm.Message
	if req.System != "" {
		messages = append(messages, llm.Message{Role: "system", Content: req.System})
	}
	messages = append(messages, llm.Message{Role: "user", Content: req.User, Images: req.Images})
	resp, err := r.Chat(context.Background(), llm.ChatRequest{Model: req.JudgeRoute, Messages: messages, CallerRole: llm.Role(req.CallerRole), Tier: llm.Tier(req.Tier), Temperature: &req.Temperature, MaxTokens: req.MaxTokens, ResponseSchema: req.Schema})
	if err != nil {
		return nil, err
	}
	return &models.Response{Content: resp.Content, Model: resp.Model, Usage: map[string]int{"prompt_tokens": resp.Usage.Input, "completion_tokens": resp.Usage.Output, "total_tokens": resp.Usage.Total}}, nil
}
