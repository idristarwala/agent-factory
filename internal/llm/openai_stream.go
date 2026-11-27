package llm

import "encoding/json"

func decodeOpenAIStream(data string) ([]StreamEvent, error) {
	var frame struct {
		Error   *json.RawMessage `json:"error"`
		Choices []struct {
			Index  int
			Finish string `json:"finish_reason"`
			Delta  struct {
				Content string
				Calls   []struct {
					Index    int
					ID       string
					Function struct{ Name, Arguments string }
				} `json:"tool_calls"`
			}
		}
		Usage *struct {
			Input         int `json:"prompt_tokens"`
			Output        int `json:"completion_tokens"`
			Total         int `json:"total_tokens"`
			OutputDetails struct {
				Reasoning int `json:"reasoning_tokens"`
			} `json:"completion_tokens_details"`
		}
	}
	if err := json.Unmarshal([]byte(data), &frame); err != nil || frame.Error != nil {
		return nil, &Error{Type: "BAD_RESPONSE", Message: "invalid or failed OpenAI stream event"}
	}
	var events []StreamEvent
	for _, choice := range frame.Choices {
		if choice.Index != 0 {
			continue
		}
		if choice.Delta.Content != "" {
			events = append(events, StreamEvent{TextDelta: choice.Delta.Content})
		}
		for _, call := range choice.Delta.Calls {
			events = append(events, StreamEvent{ToolCallIndex: call.Index, ToolCallDelta: &ToolCall{ID: call.ID, Name: call.Function.Name, Arguments: call.Function.Arguments}})
		}
		if choice.Finish != "" {
			events = append(events, StreamEvent{FinishReason: choice.Finish})
		}
	}
	if u := frame.Usage; u != nil {
		events = append(events, StreamEvent{Usage: &Usage{Input: u.Input, Output: u.Output, Total: u.Total, Reasoning: u.OutputDetails.Reasoning}})
	}
	return events, nil
}
