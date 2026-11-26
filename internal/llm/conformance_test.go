package llm

import (
	"context"
	"testing"
)

type fakeClient struct{}

func (f *fakeClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{
		Content: "ok",
		Model:   "fake",
		Usage:   Usage{Input: 1, Output: 1, Total: 2},
	}, nil
}

func (f *fakeClient) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}

type fakeHarness struct{}

func (f *fakeHarness) Client() Client { return &fakeClient{} }

func TestConformanceBasic(t *testing.T) {
	RunConformance(t, &fakeHarness{})
}

func TestAdapterNoSetenv(t *testing.T) {
	// Ensure adapters can be created from config without relying on env mutation.
	cfg := map[string]string{
		"base_url": "http://localhost:9999",
		"api_key":  "test-key",
	}
	c, err := New("openai", cfg)
	if err != nil {
		t.Fatalf("openai adapter from config: %v", err)
	}
	if c == nil {
		t.Fatal("nil client")
	}
}
