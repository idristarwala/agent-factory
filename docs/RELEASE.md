# Releases

Run `make release` to build the goal runner (`agent-factory`), management CLI (`factory`), and HTTP server (`api`) in `dist/`. Published targets are Linux amd64, macOS amd64 and arm64, and Windows amd64. The container builds the same three programs for its target platform.

A version tag triggers the release workflow. It builds the binaries, publishes engine, worker and connector containers, records their digests and the binary filenames in the release manifest, and attaches the artifacts to the GitHub release. Ordinary pull requests run vet, race tests, the Go build, and the frontend build.

Use the fully qualified references under `images.engine`, `images.worker` and `images.connector` in `release-manifest.json`. The original `image` field refers to the engine. The engine retains the goal CLI entry point and includes the API and management CLI. The worker runs the same goal CLI as a batch job. Give it a versioned factory definition, a shared database, records and a model route. [Deployment](DEPLOYMENT.md)

The connector runs a one-shot read against an operator-configured HTTP or MCP source. Supply a source file and a pack with a read-only grant. For example, with `CONNECTOR_IMAGE` set to the pinned connector reference and a grant for the named source:

```sh
docker run --rm -v "$PWD/config:/config:ro" -e FACTORY_SOURCE_CONFIG=/config/sources.yaml \
  -e FACTORY_PACK=review -e RECORDS_API_TOKEN "$CONNECTOR_IMAGE" records cursor=next
```

The process writes the source JSON to stdout for a calling harness. Use the same pinned reference in a connector Dockerfile's `FROM` instruction to add a custom program and override its entry point. [Source and grant configuration](HTTP_TOOLS.md)

The release manifest also identifies the exact schema sources shipped with the engine. Compare `pack_spec_version` to detect changes in the published pack schema or its referenced rule schema. Compare `database_schema_version` to detect changes in the shared PostgreSQL and SQLite migration source, including the dialect-specific primary keys. Each value is a SHA-256 digest of the compact, sorted JSON map from repository path to file SHA-256. Formatting edits also change these identifiers. Equal versions mean identical sources. Compatibility and migration safety require separate review.

Engine releases carry the project’s proprietary license. The release assets include `LICENSE`, the release manifest names that file, and the runtime image includes the same text at `/usr/share/licenses/agent-factory/LICENSE`. [License and permission inquiries](../README.md#license)

## Publish a proposed pack revision

Run the two commands below in your pack repository. The first reads the pack, fixtures and referenced definitions from the same Git commit, runs the fixture suite offline and prints a JSON receipt only after all fixtures pass. Working-tree edits are exercised by plain `factory test`. For a custom CI suite, write a JSON receipt with `status` set to `passed`, the full Git `commit`, the lowercase hex `pack_sha256` of the exact pack file, and a nonempty `tests` array naming the checks. Additional evidence fields are retained. The publisher trusts the CI producer’s test report. It verifies the commit and pack digest against the checkout.

```sh
factory test --receipt packs/support/pack.yaml > test-evidence.json
factory publish -pack packs/support/pack.yaml -evidence test-evidence.json -expected 0
```

The first publication uses expected revision zero. Use the latest revision for a subsequent change. A retry of the same publication returns the existing revision. Conflicting writes fail. The command prints the saved revision as JSON. Its data contains the commit, the SHA-256 digests, the original pack and evidence bytes (base64), and the status `proposed`. Publishing leaves activation to review.

The CLI and API must share the SQLite file or PostgreSQL database and the same factory scope. Process-local memory cannot retain a publication after CLI exit. Fetch the immutable history with `GET /api/releases/{pack-name}`. [Storage configuration](CONFIGURATION.md)

## Review a workspace pack

Read the proposed revision with `GET /api/releases/{pack-name}`. Inspect its commit, pack bytes, SHA-256 and CI receipt. An administrator can then send the exact proposal revision and a decision to `POST /api/releases/{pack-name}/review`.

```json
{"expected":1,"decision":"approve","note":"Reviewed the fixture evidence for this commit."}
```

Use `reject` to record a rejection. A successful review appends a human-authored revision with the original bytes and a `proposal_revision` reference. Repeating the same decision and note while it is the latest review returns the existing revision. A stale or conflicting edit returns HTTP 409.

Deployment harnesses can fetch `GET /api/releases/{pack-name}/active` with viewer access to obtain the most recently approved snapshot. Before the first approval this returns HTTP 404. A newer proposal or rejection leaves the previous approval available. Pin the returned version and digests when deploying the pack. Workspace pack approval does not publish an engine tag or change swarm execution mode.

### Use an approved pack in a goal

Set `"pack_release":{"name":"review","version":2}` in a versioned factory definition, using the approval document's version. The goal CLI and workspace API load that exact reviewed pack when preparing the run, check its evidence and digests, and require a compatible worker role and record type. Configured browser and proposal tools must also be allowed by the pack and separately granted by the workspace. Active pack policies accompany the workspace policies sent to the worker and judge. Workspace rule selections leave the pack policies in place. Give the selected workspace rules and the pack rules distinct IDs.

The saved plan and each CLI result include `approved_pack` with its name, review revision, commit and digests. To adopt another approval, save a new factory definition revision. Existing definitions without a pack reference keep their existing rule selection.

### Promote a reviewed observation

A workspace run using a pinned pack starts in observe mode. Workers and judges produce proposals, and a human can review each one. The run retains the original record values. Inspect its decisions, calls and costs before promoting the factory.

To permit subsequent approved changes to update run records, save a new revision of the same factory definition with `"pack_promotion":{"run":"RUN_ID","version":N}`. Use the exact revision of a completed observation containing a human-approved decision. Keep the other definition fields and the approved pack revision identical to the observed plan. A changed definition or pack needs its own observation. Existing queued runs retain the settings they pinned when they started. The workspace-wide `observe: true` setting continues to take precedence. Human review and tool grants still govern record changes.
