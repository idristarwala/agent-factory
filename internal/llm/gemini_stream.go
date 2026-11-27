package llm

import "encoding/json"

type geminiStreamState struct{ nextCall int }

func (s *geminiStreamState) decode(data string) ([]StreamEvent, error) {
	var frame struct {
		Error      *json.RawMessage
		Candidates []struct {
			Index   int
			Finish  string `json:"finishReason"`
			Content struct {
				Parts []struct {
					Text             string
					ThoughtSignature string
					FunctionCall     *struct {
						Name string
						Args json.RawMessage
					}
				}
			}
		}
		Usage *struct {
			Input     int `json:"promptTokenCount"`
			Output    int `json:"candidatesTokenCount"`
			Total     int `json:"totalTokenCount"`
			Reasoning int `json:"thoughtsTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal([]byte(data), &frame); err != nil || frame.Error != nil {
		return nil, &Error{Type: "BAD_RESPONSE", Message: "invalid or failed Gemini stream event"}
	}
	var events []StreamEvent
	for _, c := range frame.Candidates {
		if c.Index != 0 {
			continue
		}
		for _, p := range c.Content.Parts {
			if p.Text != "" {
				events = append(events, StreamEvent{TextDelta: p.Text})
			}
			if f := p.FunctionCall; f != nil {
				args := string(f.Args)
				if args == "" {
					args = "{}"
				}
				events = append(events, StreamEvent{ToolCallIndex: s.nextCall, ToolCallDelta: &ToolCall{ID: f.Name, Name: f.Name, Arguments: args, ThoughtSignature: p.ThoughtSignature}})
				s.nextCall++
			}
		}
		if c.Finish != "" {
			events = append(events, StreamEvent{FinishReason: c.Finish})
		}
	}
	if u := frame.Usage; u != nil {
		events = append(events, StreamEvent{Usage: &Usage{Input: u.Input, Output: u.Output, Total: u.Total, Reasoning: u.Reasoning}})
	}
	return events, nil
}
