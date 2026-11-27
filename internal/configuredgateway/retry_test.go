package configuredgateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/idristarwala/agent-factory/internal/llm"
)

func TestRetryAndRecordAttempts(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(429)
			return
		}
		fmt.Fprint(w, `{"model":"served","choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":2,"completion_tokens":3}}`)
	}))
	defer server.Close()
	records := llm.NewRecorder()
	gateway := retryClient{inner: &llm.RecordingClient{Inner: llm.NewOpenAIAdapter(server.URL, "test", nil), Recorder: records}}
	start := time.Now()
	resp, err := gateway.Chat(t.Context(), llm.ChatRequest{Model: "requested"})
	if err != nil || resp.Content != "ok" {
		t.Fatal(resp, err)
	}
	if time.Since(start) < time.Second || len(records.Records) != 2 {
		t.Fatal("missing delay or attempt")
	}
	first, last := records.Records[0], records.Records[1]
	if first.Status != 429 || first.Model != "requested" || first.Error != "RATE_LIMITED" || last.Model != "served" || last.Status != 200 || last.Input != 2 || last.Output != 3 || last.Total != 5 || last.Latency <= 0 {
		t.Fatalf("wrong records: %+v", records.Records)
	}
}

func TestRetryLimits(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		delay    time.Duration
		attempts int
	}{
		{"exhausted", 503, 0, 3}, {"long_delay", 429, time.Minute, 1}, {"forbidden", 403, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := 0
			_, err := withRetry(t.Context(), func() (int, error) {
				n++
				return 0, &llm.Error{Type: "test", Status: tc.status, Retryable: tc.status != 403, RetryAfter: tc.delay}
			})
			var failure *llm.Error
			if n != tc.attempts || !errors.As(err, &failure) || failure.Status != tc.status {
				t.Fatalf("%d attempts, error %v", n, err)
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	n := 0
	_, err := withRetry(ctx, func() (int, error) { n++; cancel(); return 0, &llm.Error{Status: 429, Retryable: true} })
	if n != 1 || !errors.Is(err, context.Canceled) {
		t.Fatal(n, err)
	}
}

func TestStreamRecordsFinalUsageOnce(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: "+`{"choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`+"\n\ndata: "+`{"usage":{"prompt_tokens":2,"completion_tokens":4,"total_tokens":6}}`+"\n\ndata: [DONE]\n\n")
	}))
	defer s.Close()
	records := llm.NewRecorder()
	gw := llm.RecordingClient{Inner: llm.NewOpenAIAdapter(s.URL, "key", nil), Recorder: records}
	stream, err := gw.Stream(t.Context(), llm.ChatRequest{Model: "streaming-model"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := llm.CollectStream(stream); err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if len(records.Records) != 1 || records.Records[0].Total != 6 || records.Records[0].Output != 4 {
		t.Fatal(records.Records)
	}
}
