# Getting started (15 minutes)

Run [the service-desk example](RUNNING.md) from a checkout to evaluate a real record. A model gateway is required for evaluation. To inspect the sample pack and run fixture tests without provider credentials, run these commands from the repository root.

```sh
go run ./cmd/factory validate examples/starter/factory.yaml
go run ./cmd/factory test
```

Pass pack files, pack directories or a `factory.yaml` manifest. Manifest pack paths are relative to that manifest. A missing file, malformed pack or schema violation returns a nonzero exit code. With no arguments the CLI discovers local manifests and pack files, falling back to the repository starter. Validation covers the listed packs. Use the corresponding runtime checks for gateway and tool configuration. [Pack schema](../schemas/pack.schema.json) · [Releases](RELEASE.md)

## CI examples

GitHub Actions and GitLab CI examples that invoke `factory validate` and `factory test` live at the repository root (`.github/workflows/` and `.gitlab-ci.yml`). Both exit non-zero on failure and need no provider credentials.

## Pack format

Pack rules default to active when `active` is omitted. Set `active: false` to keep a rule disabled. See `schemas/pack.schema.json` for the full contract. A minimal pack looks like:

```yaml
name: catalog
description: Improve product titles while preserving seller wording.
node_types: [product]
worker_roles: [title_fixer]
rules:
  - id: keep_original_wording
    text: Keep the original seller wording as much as possible.
    active: true
tools:
  - catalog.get_item
```

## Start a local Compose deployment

Run `factory up` in the project checkout with Docker Compose v2 installed and the Docker daemon running. The CLI runs `docker compose up --detach --wait`. It waits for configured health checks and for other services to start. Repeated calls reconcile the same Compose project. A failed build, startup, or health check returns a nonzero status.

Use `factory up -f path/to/compose.yaml` to select a deployment explicitly. Otherwise, an exported `COMPOSE_FILE` or a Compose file at the project root uses Docker’s standard discovery and override behavior. A starter repository with only `deploy/compose.yaml` uses that file. To inspect or stop the services, run `docker compose ps` or `docker compose down` with the same file selection.

The repository Compose project starts the API and PostgreSQL and checks `/health`. Export the credentials for your selected model provider before startup. For a file-configured gateway, mount the configuration and referenced files under `./secrets` and set `LLM_GATEWAY_CONFIG` to its container path under `/etc/secrets`.

## Run pack fixtures in your own repository

Run `factory test` alongside your `factory.yaml`, or pass pack files, directories or manifests explicitly, for example `factory test examples/service-desk`. Each pack needs at least one `fixtures/*.json` file. A compiled CLI is sufficient in your repository. In an engine checkout the no-argument command also runs the Go contract suites.

A fixture supplies a goal, a single source record, and a `definition` path relative to the pack directory. The definition selects a declared worker role and defines editable fields. The runner loads the pack rules and executes the workspace using an isolated in-process store. `worker_reply` and `judge_reply` are scripted model outputs. Author the expected field changes in `expected_patch` and the expected judge verdict in `expected_approved`. Invalid fixtures, missing fixtures, wrong verdicts, and mismatched patches return a nonzero exit status. These checks need no provider credentials and exercise the worker schema, judge and review path. Live-model behavior is tested separately. [Service-desk fixture](../examples/service-desk/fixtures/routing.json) · [Factory definition](../examples/service-desk/factory.json)

## Validate rule files

Run `factory validate --rule rules/*.yaml` to check individual rules in CI. The command accepts YAML and JSON. Each file contains one rule, for example:

```yaml
id: preserve-source
text: Keep the exact identifier supplied by the source.
active: true
```

The CLI embeds the same schema used for entries in a pack’s `rules` array. Validation works offline and exits nonzero on the first invalid file. To use a validated rule in a pack, include its object in the pack’s `rules` array. [Rule schema](../schemas/rule.schema.json)
