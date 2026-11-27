package llm

import (
	"context"
	"testing"
	"time"
)

type keyHolder struct {
	APIKey string
}

func (k *keyHolder) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: k.APIKey}, nil
}
func (k *keyHolder) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}
func (k *keyHolder) SetAPIKey(key string) { k.APIKey = key }

func TestTokenRefresherInjectsKey(t *testing.T) {
	holder := &keyHolder{APIKey: "old"}
	calls := 0
	ref := NewTokenRefresher(holder, func(ctx context.Context) (string, error) {
		calls++
		return "fresh-key", nil
	}, time.Hour)

	resp, err := ref.Chat(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "fresh-key" {
		t.Errorf("expected injected key, got %q", resp.Content)
	}
	if calls != 1 {
		t.Errorf("expected 1 refresh, got %d", calls)
	}

	// Second call should not refresh
	_, err = ref.Chat(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("expected still 1 refresh, got %d", calls)
	}
}
