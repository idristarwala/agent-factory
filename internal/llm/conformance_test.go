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
