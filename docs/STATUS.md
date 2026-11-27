# Current Status

## Fixed
- MaxTokens raised to 16000 for reasoning models
- GuardDecision forces Approved=false on worker failure
- Missing imports fixed (images_evidence, yaml)
- Context recipe wired into worker prompt and mutation evidence
- Basic YAML helpers and BuildResponseSchema added
- Postgres store and factory CLI improvements in place

## Remaining minor
- Full YAML library (optional)
- Regenerate go.sum locally after go get github.com/lib/pq
- Schema now wired for OpenAI (json_schema) and Gemini (responseSchema)

All high-impact items are complete.
