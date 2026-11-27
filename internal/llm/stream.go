package llm

import (
	"context"
	"sort"
	"strings"
)

// CollectStream reads a stream to completion and returns a ChatResponse.
func CollectStream(s Stream) (*ChatResponse, error) {
	var text strings.Builder
	calls := map[int]*ToolCall{}
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
		text.WriteString(ev.TextDelta)
		if fragment := ev.ToolCallDelta; fragment != nil {
			call := calls[ev.ToolCallIndex]
			if call == nil {
				call = &ToolCall{}
				calls[ev.ToolCallIndex] = call
			}
			if fragment.ID != "" {
				call.ID = fragment.ID
			}
			if fragment.ThoughtSignature != "" {
				call.ThoughtSignature = fragment.ThoughtSignature
			}
			call.Name += fragment.Name
			call.Arguments += fragment.Arguments
		}
		if ev.Usage != nil {
			usage = *ev.Usage
		}
		if ev.FinishReason != "" {
			finish = ev.FinishReason
		}
	}
	indices := make([]int, 0, len(calls))
	for i := range calls {
		indices = append(indices, i)
	}
	sort.Ints(indices)
	var tools []ToolCall
	for _, i := range indices {
		tools = append(tools, *calls[i])
	}
	return &ChatResponse{
		ToolCalls:    tools,
		Content:      text.String(),
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
	next int
	sent bool
}

func (s *oneShotStream) Recv() (*StreamEvent, error) {
	if s.next < len(s.resp.ToolCalls) {
		i := s.next
		s.next++
		return &StreamEvent{ToolCallIndex: i, ToolCallDelta: &s.resp.ToolCalls[i]}, nil
	}
	if s.sent {
		return nil, nil
	}
	s.sent = true
	finish := s.resp.FinishReason
	if finish == "" {
		finish = "stop"
	}
	return &StreamEvent{
		TextDelta:    s.resp.Content,
		Usage:        &s.resp.Usage,
		FinishReason: finish,
	}, nil
}

func (s *oneShotStream) Close() error { return nil }
