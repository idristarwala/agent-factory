package llm

import (
	"context"
	"time"
)

// Tier is the logical quality tier agents request.
type Tier string

const (
	TierCheap    Tier = "cheap"
	TierFrontier Tier = "frontier"
	TierCouncil  Tier = "council"
)

// Role of the caller. Used for tier policy enforcement.
type Role string

const (
	RoleWorker  Role = "worker"
	RoleSteward Role = "steward"
	RoleJudge   Role = "judge"
)

// Message is a single turn in a conversation.
type Message struct {
	Role       string // "system", "user", "assistant", "tool"
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

// ToolSpec describes a tool the model may call.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON schema
}

// ToolCall is a model-requested tool invocation.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON
}

// ChatRequest is the normalized request every adapter receives.
type ChatRequest struct {
	Model          string // logical name or gateway ID; router resolves
	Tier           Tier
	Messages       []Message
	Tools          []ToolSpec
	Temperature    *float64
	MaxTokens      int
	ResponseSchema map[string]any // optional JSON Schema for a complete chat response
	ResponseFormat string         // "text" or "json"
	Metadata       map[string]string
	CallerRole     Role // set by router for policy
}

// Usage is normalized token accounting.
type Usage struct {
	Input     int
	Output    int
	Reasoning int
	Total     int
}

// ChatResponse is the normalized response.
type ChatResponse struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
	Model        string // actual gateway model ID served
	LogicalModel string // logical name requested
	Usage        Usage
}

// Client is the interface every gateway adapter must implement.
type Client interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	// Stream is optional; return ErrStreamingUnsupported if not implemented.
	Stream(ctx context.Context, req ChatRequest) (Stream, error)
}

// Stream yields incremental events.
type Stream interface {
	Recv() (*StreamEvent, error)
	Close() error
}

// StreamEvent is one chunk from a stream.
type StreamEvent struct {
	TextDelta     string
	ToolCallDelta *ToolCall
	Usage         *Usage
	FinishReason  string
	Err           error
}

var ErrStreamingUnsupported = &Error{Type: "CAPABILITY", Message: "streaming not supported"}

// Error is the typed error every adapter returns.
type Error struct {
	Type       string // TIMEOUT, TRANSPORT, RATE_LIMITED, GATEWAY_5XX, AUTH, BAD_REQUEST, BAD_RESPONSE, CAPABILITY, CONTENT_FILTERED
	Retryable  bool
	Status     int
	RetryAfter time.Duration
	Message    string
	Cause      error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Type + ": " + e.Message + ": " + e.Cause.Error()
	}
	return e.Type + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

func IsRetryable(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Retryable
	}
	return false
}
