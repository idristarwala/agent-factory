package llm

import (
	"context"
	"time"
)

// RecordingClient wraps a Client and records usage on every call.
type RecordingClient struct {
	Inner Client
}

func (c *RecordingClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	start := time.Now()
	resp, err := c.Inner.Chat(ctx, req)
	status := "ok"
	if err != nil {
		if e, ok := err.(*Error); ok {
			status = e.Type
		} else {
			status = "error"
		}
		RecordUsage(req.Model, time.Since(start), Usage{}, status)
		return nil, err
	}
	RecordUsage(resp.Model, time.Since(start), resp.Usage, status)
	return resp, nil
}

func (c *RecordingClient) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return c.Inner.Stream(ctx, req)
}
