# Release Pipeline (Issue #9)

## Current state

- `cmd/factory` CLI skeleton
- `deploy/compose.yaml` for local runs
- Helm values and basic chart structure
- Pack loader and validation

## Next steps for full release

1. Tag a version (`git tag v0.1.0`)
2. Build and push container images pinned by digest
3. Generate a release manifest listing digests and schema version
4. Starter repo template that pins the engine version

The hourly automation continues advancing these artifacts.
