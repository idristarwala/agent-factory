# Anthropic Schema Enforcement

Anthropic has no native response_schema mode.

`AnthropicForcedTool` builds a forced tool call whose `input_schema` is the reply schema.

The Anthropic client should:
1. Call `AnthropicForcedTool(req.Schema)` when schema is present.
2. Add the tool to the request `tools` array.
3. Set `tool_choice` to force the tool.
4. Extract the tool input from the response as the structured content.

This closes the remaining #14 adapter gap.
