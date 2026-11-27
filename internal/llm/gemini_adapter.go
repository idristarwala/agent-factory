package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

func (a *GeminiAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	parts := []map[string]any{{"text": req.Messages[len(req.Messages)-1].Content}}
	body := map[string]any{
		"contents": []map[string]any{{"parts": parts}},
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
				body["generationConfig"] = map[string]any{"thinkingConfig": rb}
			}
		}
	}
	b, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.BaseURL+"/models/"+req.Model+":generateContent?key="+a.APIKey, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range a.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := a.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, &Error{Type: "GATEWAY", Status: resp.StatusCode, Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429, Message: fmt.Sprintf("status %d", resp.StatusCode)}
	}

	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text"`
					FunctionCall *struct {
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
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
					ID:        part.FunctionCall.Name,
					Name:      part.FunctionCall.Name,
					Arguments: string(args),
				})
			}
		}
	}
	return &ChatResponse{Content: content, ToolCalls: toolCalls}, nil
}

func (a *GeminiAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	// Basic streaming: set streamGenerateContent and parse SSE text deltas.
	// Falls back to one-shot if the endpoint does not support it.
	body := map[string]any{
		"contents": []map[string]any{{
			"role": "user",
			"parts": []map[string]any{{"text": req.Messages[len(req.Messages)-1].Content}},
		}},
		"generationConfig": map[string]any{
			"maxOutputTokens": req.MaxTokens,
		},
	}
	b, _ := json.Marshal(body)
	url := a.BaseURL + "/v1beta/models/" + req.Model + ":streamGenerateContent?alt=sse"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-goog-api-key", a.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range a.Headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := a.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, &Error{Type: "GATEWAY", Status: resp.StatusCode, Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429}
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
