package llm

import (
	"context"
	"testing"
)

type routerFakeClient struct{}

func (f *routerFakeClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: "ok"}, nil
}
func (f *routerFakeClient) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}

func TestRouterRejectsWorkerFrontier(t *testing.T) {
	r := NewRouter(&routerFakeClient{})
	_, err := r.Chat(context.Background(), ChatRequest{
		Tier:       TierFrontier,
		CallerRole: RoleWorker,
		Messages:   []Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error for worker requesting frontier")
	}
	if e, ok := err.(*Error); !ok || e.Type != "CAPABILITY" {
		t.Fatalf("expected CAPABILITY error, got %v", err)
	}
}

func TestRouterAllowsJudgeFrontier(t *testing.T) {
	r := NewRouter(&routerFakeClient{})
	resp, err := r.Chat(context.Background(), ChatRequest{
		Tier:       TierFrontier,
		CallerRole: RoleJudge,
		Messages:   []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestRouterAllowsWorkerCheap(t *testing.T) {
	r := NewRouter(&routerFakeClient{})
	_, err := r.Chat(context.Background(), ChatRequest{
		Tier:       TierCheap,
		CallerRole: RoleWorker,
		Messages:   []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
}
