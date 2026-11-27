package llm

import (
	"io"
	"strings"
	"testing"
)

func TestCollectStreamedToolResults(t *testing.T) {
	for _, tc := range []struct {
		name, frames, finish string
		decode               func(string) ([]StreamEvent, error)
	}{
		{"openai", `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"first","function":{"name":"lookup","arguments":"{"}},{"index":1,"id":"second","function":{"name":"read","arguments":"{}"}}]}}]}
data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"}"}}]},"finish_reason":"tool_calls"}]}
data: {"choices":[],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}
data: [DONE]`, "tool_calls", decodeOpenAIStream},
		{"anthropic", `data: {"type":"message_start","message":{"usage":{"input_tokens":2}}}
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","name":"lookup","id":"first"}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{}"}}
data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","name":"read","id":"second"}}
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{}"}}
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":3}}
data: {"type":"message_stop"}`, "tool_use", new(anthropicStreamState).decode},
		{"gemini", `data: {"candidates":[{"content":{"parts":[{"functionCall":{"name":"lookup","args":{}}},{"functionCall":{"name":"read","args":{}}}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":3,"totalTokenCount":5}}`, "STOP", new(geminiStreamState).decode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stream := &sseStream{r: io.NopCloser(strings.NewReader(tc.frames)), decode: tc.decode}
			defer stream.Close()
			result, err := CollectStream(stream)
			if err != nil {
				t.Fatal(err)
			}
			if result.FinishReason != tc.finish || result.Usage != (Usage{Input: 2, Output: 3, Total: 5}) || len(result.ToolCalls) != 2 {
				t.Fatalf("unexpected result %+v", result)
			}
			for i, name := range []string{"lookup", "read"} {
				call := result.ToolCalls[i]
				if call.Name != name || call.ID == "" || call.Arguments != "{}" {
					t.Fatalf("unexpected call %+v", call)
				}
			}
		})
	}
}
