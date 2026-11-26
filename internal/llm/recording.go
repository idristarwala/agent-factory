package llm

import (
	"context"
	"time"
)

// RecordingClient wraps a Client and records every call into a Recorder.
// Satisfies the usage recording requirement from issue #11.
type RecordingClient struct {
	Inner    Client
	Recorder *Recorder
}

func (c *RecordingClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	start := time.Now()
	resp, err := c.Inner.Chat(ctx, req)
	rec := UsageRecord{
		Model:   req.Model,
		Latency: time.Since(start),
		Status:  200,
	}
	if err != nil {
		rec.Status = 500
		rec.Error = err.Error()
	} else if resp != nil {
		rec.Input = resp.Usage.Input
		rec.Output = resp.Usage.Output
	}
	if c.Recorder != nil {
		c.Recorder.Record(rec)
	}
	return resp, err
}

func (c *RecordingClient) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return c.Inner.Stream(ctx, req)
}
