# Project status

Open issues track work and its acceptance criteria. [Issue tracker](https://github.com/idristarwala/agent-factory/issues)

| Area | Verification and open work |
|---|---|
| Runtime and learning | `go test ./internal/swarm ./internal/learning ./internal/regression` covers worker failures, learning across rounds and replay gating. The original field evaluation requires the source dataset and gateway. [Field test](https://github.com/idristarwala/agent-factory/issues/10) |
| Gateways and tools | Adapter conformance and tool tests run against local fixtures. Enterprise gateway validation and evidence coverage have separate field requirements. [Gateway](https://github.com/idristarwala/agent-factory/issues/11) · [Tools](https://github.com/idristarwala/agent-factory/issues/12) |
| Operator UI | `make ui-build` installs the locked dependencies and builds the app. The UI epic tracks the individual screens and runtime integration. [Operator UI](https://github.com/idristarwala/agent-factory/issues/34) |
| Packaging | Release artifacts and supported platforms are listed in the [release guide](RELEASE.md). Packaging acceptance is tracked in [issue 9](https://github.com/idristarwala/agent-factory/issues/9). |

Run `make ci` before pushing. Install the repository hook with `git config core.hooksPath .githooks`. The Go module and its checksums are committed.
