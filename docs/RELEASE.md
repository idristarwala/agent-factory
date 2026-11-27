# Releases

Run `make release` to build the goal runner (`agent-factory`), management CLI (`factory`), and HTTP server (`api`) in `dist/`. Published targets are Linux amd64, macOS amd64 and arm64, and Windows amd64. The container builds the same three programs for its target platform.

A version tag triggers the release workflow. It builds the binaries, publishes the container, records its digest and binary filenames in the release manifest, and attaches the artifacts to the GitHub release. Ordinary pull requests run vet, race tests, the Go build, and the frontend build.

## Publish a proposed pack revision

Run your checks on the committed pack in your user repository. After they pass, have CI write a JSON receipt with `status` set to `passed`, the full Git `commit`, the lowercase hex `pack_sha256` of the exact pack file, and a nonempty `tests` array naming the checks. Additional evidence fields are retained. The publisher trusts the CI producer’s test report. It verifies the commit and pack digest against the checkout.

```sh
factory publish -pack packs/support/pack.yaml -evidence test-evidence.json -expected 0
```

The first publication uses expected revision zero. Use the latest revision for a subsequent change. A retry of the same publication returns the existing revision. Conflicting writes fail. The command prints the saved revision as JSON. Its data contains the commit, the SHA-256 digests, the original pack and evidence bytes (base64), and the status `proposed`. Publishing leaves activation to review.

The CLI and API must share the SQLite file or PostgreSQL database and the same factory scope. Process-local memory cannot retain a publication after CLI exit. Fetch the immutable history with `GET /api/releases/{pack-name}`. [Storage configuration](CONFIGURATION.md)
