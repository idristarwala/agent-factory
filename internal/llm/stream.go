package llm

import "context"

// CollectStream reads a stream to completion and returns a ChatResponse.
func CollectStream(s Stream) (*ChatResponse, error) {
	var text string
	var usage Usage
	var finish string
	for {
		ev, err := s.Recv()
		if err != nil {
			return nil, err
		}
		if ev == nil {
			break
		}
		if ev.Err != nil {
			return nil, ev.Err
		}
		text += ev.TextDelta
		if ev.Usage != nil {
			usage = *ev.Usage
		}
		if ev.FinishReason != "" {
			finish = ev.FinishReason
		}
		if finish != "" {
			break
		}
	}
	return &ChatResponse{
		Content:      text,
		FinishReason: finish,
		Usage:        usage,
	}, nil
}

// ChatAsStream wraps a non-streaming Client so callers can always use Stream.
type ChatAsStream struct {
	Client Client
}

func (c *ChatAsStream) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return c.Client.Chat(ctx, req)
}

func (c *ChatAsStream) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	resp, err := c.Client.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	return &oneShotStream{resp: resp}, nil
}

type oneShotStream struct {
	resp *ChatResponse
	sent bool
}

func (s *oneShotStream) Recv() (*StreamEvent, error) {
	if s.sent {
		return nil, nil
	}
	s.sent = true
	return &StreamEvent{
		TextDelta:    s.resp.Content,
		Usage:        &s.resp.Usage,
		FinishReason: "stop",
	}, nil
}

func (s *oneShotStream) Close() error { return nil }
