package configuredgateway

import (
	"context"
	"errors"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/models"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	policy, _ := (retryOverrides{}).resolve()
	gateway := retryClient{policy: policy, inner: &llm.RecordingClient{Inner: llm.NewOpenAIAdapter(server.URL, "test", nil), Recorder: records}}
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
	policy, _ := (retryOverrides{}).resolve()
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
			_, err := withRetry(t.Context(), policy, func() (int, error) {
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
	_, err := withRetry(ctx, policy, func() (int, error) { n++; cancel(); return 0, &llm.Error{Status: 429, Retryable: true} })
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

func TestRetryOverridesAtGatewayEntryPoint(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(503) }))
	defer server.Close()
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_BASE_URL", server.URL)
	t.Setenv("LLM_API_KEY", "fixture")
	t.Setenv("LLM_JUDGE_PROVIDER", "")
	t.Setenv("LLM_JUDGE_MODEL", "")
	t.Setenv("LLM_GATEWAY_CONFIG", "")
	t.Setenv("LLM_RETRY_ATTEMPTS", "1")
	t.Setenv("LLM_RETRY_BASE_DELAY", "1ms")
	t.Setenv("LLM_RETRY_MAX_DELAY", "2ms")
	check := func(want int32) {
		t.Helper()
		calls.Store(0)
		g, err := NewFromEnv()
		if err != nil {
			t.Fatal(err)
		}
		_, err = g.Generate(models.Request{CallerRole: "worker", Tier: models.TierCheap})
		var failure *llm.Error
		if !errors.As(err, &failure) || failure.Status != 503 || calls.Load() != want {
			t.Fatal(err, calls.Load())
		}
	}
	check(1)
	t.Setenv("LLM_RETRY_ATTEMPTS", "17")
	if _, err := NewFromEnv(); err == nil {
		t.Fatal("invalid environment policy accepted")
	}
	yaml := fmt.Sprintf("gateways:\n  lab: {base_url: %s, auth: {type: none}}\nmodels:\n  cheap: {gateway: lab, protocol: openai, model: fixture, retry: {attempts: 2, base_delay: 1ms, max_delay: 2ms}}\n", server.URL)
	file := filepath.Join(t.TempDir(), "routes.yaml")
	if err := os.WriteFile(file, []byte(yaml), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLM_GATEWAY_CONFIG", file)
	check(2)
	if err := os.WriteFile(file, []byte(strings.Replace(yaml, "max_delay: 2ms", "max_delay: 0s", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFromEnv(); err == nil {
		t.Fatal("invalid route policy accepted")
	}
}
