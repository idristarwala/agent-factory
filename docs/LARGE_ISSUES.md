# Large Design Issues Status

## #1–#8 Pluggable LLM Gateways

Foundation complete in `internal/llm`:
- Client / ChatRequest / ChatResponse / Message / ToolSpec / ToolCall / Usage
- Adapter registry with Register/New
- Typed Error with IsRetryable
- Router and basic streaming types
- Basic conformance helper

Remaining for full close:
- Complete conformance suite covering all cases in #8
- Rate limiting
- Full context cancellation wiring in every call path
- Example out-of-tree adapter
- Declarative config loading

## #9 Packaging as Installable Product

Started:
- `cmd/factory` CLI skeleton with validate/test/readiness commands
- `internal/pack.Load` for file-based packs

Remaining:
- Full CLI implementation
- Packs replace Go domain packs
- Tool model with side-effect classes
- Observe mode
- Release pipeline and starter template
- Docker/k8s production packaging

These are tracked as ongoing work. The core factory loop is functional; these issues make it a distributable product.
