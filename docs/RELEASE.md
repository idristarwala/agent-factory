# Release Pipeline (Issue #9)

## Current state

- `cmd/factory` CLI skeleton (`up`, `validate`, `test`, `readiness`, `publish`)
- `deploy/release.sh` builds binaries and emits a sample release manifest (engine + worker digests)
- CI workflow runs `go vet`, `go test -race`, and a release-skeleton job on main
- `go.sum` committed; clean clones build
- Docker / compose / basic k8s manifests present
- Starter template: `examples/starter/factory.yaml` pins engine version + tools + gateway

## Next steps for full digest-pinned release

1. Automate `docker build` + `docker inspect` digest capture in CI (see release-skeleton job note)
2. Publish a release manifest (digests, pack-spec version, schema version) as a GitHub release asset
3. Starter repo template that pins the engine version via `factory.yaml` (template added)
4. Durable store contract tests shared by memory and SQL backends

The pipeline skeleton and starter config are ready for the remaining automation.
