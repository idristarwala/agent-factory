package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// OpenAIAdapter implements Client for OpenAI-compatible endpoints.
// Constructs and POSTs the request body without os.Setenv.
type OpenAIAdapter struct {
	Envelope *OpenAIEnvelope
	BaseURL  string
	APIKey   string
	Client   *http.Client
	Headers  map[string]string
}

func NewOpenAIAdapter(baseURL, apiKey string, headers map[string]string) *OpenAIAdapter {
	return &OpenAIAdapter{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client:  &http.Client{},
		Headers: headers,
	}
}

func (a *OpenAIAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	schema, err := compileOutputSchema(req.ResponseSchema)
	if err != nil {
		return nil, err
	}

	body, err := a.requestBody(req, false)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)
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
		Model   string `json:"model"`
		Choices []struct {
			Finish  string `json:"finish_reason"`
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := decodeReply(ctx, resp, &out); err != nil {
		return nil, err
	}
	content := ""
	var toolCalls []ToolCall
	if len(out.Choices) == 0 {
		return nil, &Error{Type: "BAD_RESPONSE", Message: "no OpenAI choices"}
	}
	content = out.Choices[0].Message.Content
	for _, tc := range out.Choices[0].Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	if err := validateOutput(schema, content); err != nil {
		return nil, err
	}
	return &ChatResponse{
		Content:      content,
		FinishReason: out.Choices[0].Finish, Model: out.Model, LogicalModel: req.Model,
		ToolCalls: toolCalls,
		Usage: Usage{
			Input:  out.Usage.PromptTokens,
			Output: out.Usage.CompletionTokens,
			Total:  out.Usage.PromptTokens + out.Usage.CompletionTokens,
		},
	}, nil
}

func (a *OpenAIAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	if req.ResponseSchema != nil {
		return nil, ErrStreamingUnsupported
	}

	// Basic SSE streaming support for OpenAI-compatible endpoints.
	// Falls back to ErrStreamingUnsupported only if the request cannot be built.
	body, err := a.requestBody(req, true)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	for k, v := range a.Headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := gatewayResponse(a.Client, httpReq)
	if err != nil {
		return nil, err
	}

	return &sseStream{r: resp.Body, decode: decodeOpenAIStream}, nil
}

type sseStream struct {
	r        io.ReadCloser
	sc       *bufio.Scanner
	done     bool
	finished bool
	decode   func(string) ([]StreamEvent, error)
	pending  []StreamEvent
}

func (s *sseStream) Recv() (*StreamEvent, error) {
	if s.sc == nil {
		s.sc = bufio.NewScanner(s.r)
		s.sc.Buffer(make([]byte, 4096), 16<<20)
	}
	for len(s.pending) == 0 {
		if s.done {
			return nil, nil
		}
		if !s.sc.Scan() {
			s.done = true
			return nil, s.sc.Err()
		}
		line := strings.TrimSpace(s.sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			s.done = true
			if !s.finished {
				return &StreamEvent{FinishReason: "stop"}, nil
			}
			return nil, nil
		}
		events, err := s.decode(data)
		if err != nil {
			s.done = true
			return nil, err
		}
		s.pending = events
	}
	ev := s.pending[0]
	s.pending = s.pending[1:]
	if ev.FinishReason != "" {
		s.finished = true
	}
	return &ev, nil
}

func (s *sseStream) Close() error { return s.r.Close() }

// BuildOpenAIBody constructs the JSON body for an OpenAI-compatible request.
func BuildOpenAIBody(req ChatRequest) ([]byte, error) {
	return buildOpenAIBody(req, false)
}

func buildOpenAIBody(req ChatRequest, stream bool) ([]byte, error) {
	msgs := make([]map[string]any, len(req.Messages))
	for i, msg := range req.Messages {
		m := map[string]any{"role": msg.Role, "content": msg.Content}
		if len(msg.ToolCalls) > 0 {
			calls := make([]map[string]any, 0, len(msg.ToolCalls))
			for _, call := range msg.ToolCalls {
				calls = append(calls, map[string]any{"id": call.ID, "type": "function", "function": map[string]string{"name": call.Name, "arguments": call.Arguments}})
			}
			m["tool_calls"] = calls
		}
		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}
		msgs[i] = m
	}
	body := map[string]any{
		"model":    req.Model,
		"messages": msgs,
	}
	if stream {
		body["stream"] = true
		body["stream_options"] = map[string]bool{"include_usage": true}
	}
	if req.MaxTokens > 0 {
		param := MapTokenParam("openai", req.Model)
		if override := req.Metadata["token_param"]; override == "max_tokens" || override == "max_completion_tokens" {
			param = override
		}
		body[param] = req.MaxTokens
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			}
		}
		body["tools"] = tools
	}
	// Provider-enforced structured output (issue #10 / #8)
	if req.ResponseSchema != nil {
		body["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "response", "strict": true, "schema": req.ResponseSchema}}
	} else if req.ResponseFormat == "json" {
		body["response_format"] = map[string]any{"type": "json_object"}
	}
	return json.Marshal(body)
}

// SetAPIKey updates the API key (used by TokenRefresher).
func (a *OpenAIAdapter) SetAPIKey(key string) {
	a.APIKey = key
}
