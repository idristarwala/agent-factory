# Project status

Open issues track work and its acceptance criteria. [Issue tracker](https://github.com/idristarwala/agent-factory/issues)

| Area | Verification and open work |
|---|---|
| Runtime and learning | `go test ./internal/swarm ./internal/learning ./internal/regression` covers worker failures, learning across rounds and replay gating. The original field evaluation requires the source dataset and gateway. [Field test](https://github.com/idristarwala/agent-factory/issues/10) |
| Gateways and tools | Adapter conformance and tool tests run against local fixtures. Configured HTTP and MCP evidence, reviewed write proposals and image tiling are verified on main. The original enterprise endpoint still needs independent validation. [Gateway](https://github.com/idristarwala/agent-factory/issues/11) · [Tools](https://github.com/idristarwala/agent-factory/issues/12) |
| Operator UI | `make ui-build` installs the locked dependencies and builds the app. The UI epic tracks the individual screens and runtime integration. [Operator UI](https://github.com/idristarwala/agent-factory/issues/34) |
| Packaging | Release artifacts and supported platforms are listed in the [release guide](RELEASE.md). Packaging acceptance is tracked in [issue 9](https://github.com/idristarwala/agent-factory/issues/9). |

The continuous CLI processes work in an in-memory graph and queue and publishes terminal audit entries and model usage to the workspace store. Configured HTTP tools and mapped images use the evidence collector.

Pull requests also compare exported internal Go helpers against the base revision and flag newly unreachable helpers. Run `make ci` before pushing. Install the repository hook with `git config core.hooksPath .githooks`. The Go module and its checksums are committed.

The environment and YAML model routes share protocol adapters, retry handling, typed errors and per-attempt usage logs. The agent-facing request interface is in `internal/models`. Wire protocol implementations are in `internal/llm`. [Gateway configuration](ENTERPRISE_GATEWAY.md)
