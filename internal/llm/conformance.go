package llm

import (
	"context"
	"testing"
)

// RunBasicConformance is a starting point for the full suite in issue #8.
// Callers supply a Client pointed at a fake server.
func RunBasicConformance(t *testing.T, c Client) {
	t.Helper()
	ctx := context.Background()
	resp, err := c.Chat(ctx, ChatRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("basic chat failed: %v", err)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
}
