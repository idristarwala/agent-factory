# Provider-Enforced Schema (#14)

## Added
- `models.ResponseSchema` type
- `BuildResponseSchema` now takes allowed and required fields separately and sets `additionalProperties: false`

## Next
- Add the schema field to `models.Request`
- Worker builds and attaches the schema on every call
- OpenAI and Gemini clients send it in native form
- Anthropic uses a forced tool call

The helper exists. Wiring into the request path is the remaining step.
