# Tracking

## Implemented

- WorkQueue (memory + SQL stub) with claim/lease
- ContinuousRunner with parallel workers and LearningLoop
- Graph.Apply merges props (regression tested)
- Worker receives active rules in prompt
- Judge sees original entity + proposed mutation
- SmartSteward.CreateWorker returns real LLMWorker after SetRuntime
- /api/feedback creates versioned rules
- Regression tests for Apply, rules, feedback
- internal/llm package with Client, registry, router, types, streaming basics

## Remaining (tracked in issues)

- Family/sibling context for workers (#10)
- Deterministic block-then-decide grouping (#10)
- Full schema-validated structured output + retry (#10)
- go.sum committed (#9, #10)
- Full context.Context wiring through all LLM calls (#1)
- Complete conformance suite (#8)
- Packaging as factory CLI + packs-as-files (#9)
- Real BigQuery/MCP clients

Critical correctness and accuracy gaps from the field test are closed. Larger packaging and full gateway conformance remain open.
