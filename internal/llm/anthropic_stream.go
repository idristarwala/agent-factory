package llm

import "encoding/json"

// Anthropic reports input usage at message_start and cumulative output usage at message_delta.
type anthropicStreamState struct{ usage Usage }

func (s *anthropicStreamState) decode(data string) ([]StreamEvent, error) {
	var frame struct {
		Type    string
		Index   int
		Message struct {
			Usage struct {
				Input  int `json:"input_tokens"`
				Output int `json:"output_tokens"`
				Read   int `json:"cache_read_input_tokens"`
				Write  int `json:"cache_creation_input_tokens"`
			}
		}
		Usage struct {
			Output int `json:"output_tokens"`
		}
		Block struct{ Type, ID, Name, Text string } `json:"content_block"`
		Delta struct {
			Type, Text string
			Partial    string `json:"partial_json"`
			Stop       string `json:"stop_reason"`
		}
	}
	if err := json.Unmarshal([]byte(data), &frame); err != nil {
		return nil, &Error{Type: "BAD_RESPONSE", Message: "invalid Anthropic stream event"}
	}
	switch frame.Type {
	case "error":
		return nil, &Error{Type: "BAD_RESPONSE", Message: "Anthropic stream failed"}
	case "message_start":
		u := frame.Message.Usage
		s.usage = Usage{Input: u.Input + u.Read + u.Write, Output: u.Output}
	case "content_block_start":
		switch frame.Block.Type {
		case "tool_use":
			return []StreamEvent{{ToolCallIndex: frame.Index, ToolCallDelta: &ToolCall{ID: frame.Block.ID, Name: frame.Block.Name}}}, nil
		case "text":
			return []StreamEvent{{TextDelta: frame.Block.Text}}, nil
		}
	case "content_block_delta":
		switch frame.Delta.Type {
		case "input_json_delta":
			return []StreamEvent{{ToolCallIndex: frame.Index, ToolCallDelta: &ToolCall{Arguments: frame.Delta.Partial}}}, nil
		case "text_delta":
			return []StreamEvent{{TextDelta: frame.Delta.Text}}, nil
		}
	case "message_delta":
		s.usage.Output = frame.Usage.Output
		s.usage.Total = s.usage.Input + s.usage.Output
		u := s.usage
		return []StreamEvent{{FinishReason: frame.Delta.Stop, Usage: &u}}, nil
	}
	return nil, nil
}
