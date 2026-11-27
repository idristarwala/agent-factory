package models

import (
	"encoding/json"
)

// AnthropicToolSchema builds a forced tool call payload for Anthropic
// so the model must return data matching the schema.
// Addresses the remaining #14 gap for Anthropic.
func AnthropicToolSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	return map[string]any{
		"type": "tool_use",
		"name": "return_structured",
		"input_schema": schema,
	}
}

// AnthropicForcedTool returns the tools array and tool_choice for a request.
func AnthropicForcedTool(schema map[string]any) (tools []map[string]any, choice map[string]any) {
	if schema == nil {
		return nil, nil
	}
	tool := map[string]any{
		"name":         "return_structured",
		"description":  "Return the structured attribute update",
		"input_schema": schema,
	}
	tools = []map[string]any{tool}
	choice = map[string]any{
		"type": "tool",
		"name": "return_structured",
	}
	return tools, choice
}

// SchemaBytes returns the schema as JSON.
func SchemaBytes(schema map[string]any) []byte {
	b, _ := json.Marshal(schema)
	return b
}
