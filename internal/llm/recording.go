package llm

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/idristarwala/agent-factory/internal/obs"
)

// RecordingClient emits one metadata-only usage record per adapter invocation.
type RecordingClient struct {
	Inner    Client
	Recorder *Recorder
	Logger   *obs.Logger
}

func (c *RecordingClient) finish(ctx context.Context, start time.Time, rec UsageRecord, err error) {
	rec.Latency = time.Since(start)
	if err != nil {
		rec.Error = "TRANSPORT"
		rec.Status = 0
		var failure *Error
		if errors.As(err, &failure) {
			rec.Error = failure.Type
			rec.Status = failure.Status
		}
		if errors.Is(err, context.Canceled) {
			rec.Error = "CANCELED"
		}
	}
	if c.Recorder != nil {
		c.Recorder.Record(rec)
	}
	if c.Logger != nil {
		c.Logger.Info(ctx, "gateway call", map[string]any{"model": rec.Model, "latency_ms": rec.Latency.Milliseconds(), "status": rec.Status, "error_type": rec.Error, "input_tokens": rec.Input, "output_tokens": rec.Output, "total_tokens": rec.Total})
	}
}

func (c *RecordingClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	start := time.Now()
	resp, err := c.Inner.Chat(ctx, req)
	rec := UsageRecord{Model: req.Model, Status: 200}
	if resp != nil {
		if resp.Model != "" {
			rec.Model = resp.Model
		}
		rec.Input = resp.Usage.Input
		rec.Output = resp.Usage.Output
		rec.Total = resp.Usage.Total
	}
	c.finish(ctx, start, rec, err)
	return resp, err
}

func (c *RecordingClient) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	start := time.Now()
	rec := UsageRecord{Model: req.Model, Status: 200}
	stream, err := c.Inner.Stream(ctx, req)
	if err != nil {
		c.finish(ctx, start, rec, err)
		return nil, err
	}
	return &recordedStream{Stream: stream, rec: rec, emit: func(rec UsageRecord, err error) { c.finish(ctx, start, rec, err) }}, nil
}

type recordedStream struct {
	Stream
	mu   sync.Mutex
	done bool
	rec  UsageRecord
	emit func(UsageRecord, error)
}

func (s *recordedStream) observe(ev *StreamEvent, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done {
		return
	}
	if ev != nil {
		if ev.Usage != nil {
			s.rec.Input = ev.Usage.Input
			s.rec.Output = ev.Usage.Output
			s.rec.Total = ev.Usage.Total
		}
		if err == nil {
			err = ev.Err
		}
		if err == nil {
			return
		}
	}
	s.done = true
	s.emit(s.rec, err)
}

func (s *recordedStream) Recv() (*StreamEvent, error) {
	ev, err := s.Stream.Recv()
	s.observe(ev, err)
	return ev, err
}
func (s *recordedStream) Close() error { err := s.Stream.Close(); s.observe(nil, err); return err }
