package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GeminiAdapter implements Client for Gemini-style endpoints.
type GeminiAdapter struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
	Headers map[string]string
}

func NewGeminiAdapter(baseURL, apiKey string, headers map[string]string) *GeminiAdapter {
	return &GeminiAdapter{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client:  &http.Client{},
		Headers: headers,
	}
}

func geminiBody(req ChatRequest) ([]byte, error) {
	var contents []map[string]any
	var system []map[string]string
	functions := map[string]string{}
	for _, message := range req.Messages {
		role := message.Role
		var parts []map[string]any
		switch role {
		case "system":
			system = append(system, map[string]string{"text": message.Content})
			continue
		case "assistant":
			role = "model"
		case "tool":
			name := functions[message.ToolCallID]
			if name == "" {
				return nil, &Error{Type: "BAD_REQUEST", Message: "tool result must reference a previous function call"}
			}
			role = "user"
			parts = append(parts, map[string]any{"functionResponse": map[string]any{"name": name, "response": map[string]any{"result": message.Content}}})
		case "user":
		default:
			return nil, &Error{Type: "BAD_REQUEST", Message: "unsupported conversation role"}
		}
		if message.Role != "tool" && message.Content != "" {
			parts = append(parts, map[string]any{"text": message.Content})
		}
		for _, call := range message.ToolCalls {
			var args map[string]any
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil || args == nil {
				return nil, &Error{Type: "BAD_REQUEST", Message: "function arguments must be an object"}
			}
			functions[call.ID] = call.Name
			part := map[string]any{"functionCall": map[string]any{"name": call.Name, "args": args}}
			if call.ThoughtSignature != "" {
				part["thoughtSignature"] = call.ThoughtSignature
			}
			parts = append(parts, part)
		}
		if len(parts) == 0 {
			return nil, &Error{Type: "BAD_REQUEST", Message: "empty conversation turn"}
		}
		contents = append(contents, map[string]any{"role": role, "parts": parts})
	}
	if len(contents) == 0 {
		return nil, &Error{Type: "BAD_REQUEST", Message: "supply a conversation turn"}
	}
	body := map[string]any{"contents": contents}
	if len(system) > 0 {
		body["systemInstruction"] = map[string]any{"parts": system}
	}
	cfg := map[string]any{}
	if req.MaxTokens > 0 {
		cfg["maxOutputTokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		cfg["temperature"] = *req.Temperature
	}
	if req.ResponseSchema != nil {
		cfg["responseMimeType"] = "application/json"
		cfg["responseJsonSchema"] = req.ResponseSchema
	} else if req.ResponseFormat == "json" {
		cfg["responseMimeType"] = "application/json"
	}
	if len(req.Tools) > 0 {
		decls := make([]map[string]any, len(req.Tools))
		for i, t := range req.Tools {
			decls[i] = map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			}
		}
		body["tools"] = []map[string]any{{"functionDeclarations": decls}}
	}
	if budgetStr, ok := req.Metadata["reasoning_budget"]; ok {
		if budget, err := parseInt(budgetStr); err == nil {
			if rb := ReasoningBudget("gemini", budget); rb != nil {
				cfg["thinkingConfig"] = rb
			}
		}
	}
	if len(cfg) > 0 {
		body["generationConfig"] = cfg
	}
	return json.Marshal(body)
}

func (a *GeminiAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	schema, err := compileOutputSchema(req.ResponseSchema)
	if err != nil {
		return nil, err
	}

	b, err := geminiBody(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.BaseURL+"/models/"+req.Model+":generateContent", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-goog-api-key", a.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range a.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := gatewayResponse(a.Client, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Model string `json:"modelVersion"`
		Usage struct {
			Input     int `json:"promptTokenCount"`
			Output    int `json:"candidatesTokenCount"`
			Reasoning int `json:"thoughtsTokenCount"`
			Total     int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text             string `json:"text"`
					ThoughtSignature string `json:"thoughtSignature"`
					FunctionCall     *struct {
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := decodeReply(ctx, resp, &out); err != nil {
		return nil, err
	}
	content := ""
	var toolCalls []ToolCall
	if len(out.Candidates) > 0 {
		for _, part := range out.Candidates[0].Content.Parts {
			if part.Text != "" {
				content += part.Text
			}
			if part.FunctionCall != nil {
				args, _ := json.Marshal(part.FunctionCall.Args)
				toolCalls = append(toolCalls, ToolCall{
					ID:               part.FunctionCall.Name,
					ThoughtSignature: part.ThoughtSignature,
					Name:             part.FunctionCall.Name,
					Arguments:        string(args),
				})
			}
		}
	}
	if len(out.Candidates) == 0 {
		return nil, &Error{Type: "BAD_RESPONSE", Message: "no Gemini candidates"}
	}
	if err := validateOutput(schema, content); err != nil {
		return nil, err
	}
	return &ChatResponse{Content: content, ToolCalls: toolCalls, Model: out.Model, LogicalModel: req.Model, Usage: Usage{Input: out.Usage.Input, Output: out.Usage.Output, Reasoning: out.Usage.Reasoning, Total: out.Usage.Total}}, nil
}

func (a *GeminiAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	if req.ResponseSchema != nil {
		return nil, ErrStreamingUnsupported
	}

	b, err := geminiBody(req)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(a.BaseURL, "/")
	if !strings.HasSuffix(base, "/v1beta") {
		base += "/v1beta"
	}
	url := base + "/models/" + req.Model + ":streamGenerateContent?alt=sse"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-goog-api-key", a.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range a.Headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := gatewayResponse(a.Client, httpReq)
	if err != nil {
		return nil, err
	}

	return &sseStream{r: resp.Body}, nil
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// SetAPIKey updates the API key (used by TokenRefresher).
func (a *GeminiAdapter) SetAPIKey(key string) {
	a.APIKey = key
}
