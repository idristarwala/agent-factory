# Security

## Secrets

- Never commit secrets.
- Use the centralized loader (`SECRETS_DIR`).
- Mount secrets read-only in Docker and Kubernetes.
- Rotate by replacing the file; the loader re-reads on mtime change.

## Auth methods

File-configured LLM routes reference key files. Environment-only routes take their provider credentials from the process environment. The API uses its own bearer token. [Configuration and precedence](CONFIGURATION.md) · [Authentication guide](../AUTH.md)

## Judge fail-closed

If the model errors or returns an unclear reply, the proposal is rejected. No silent approvals.

## Network

- Outbound calls only to configured LLM endpoints and tool backends.
- The Compose stack binds the API to host loopback on port 8080 and publishes PostgreSQL on port 5432. Configure network exposure and `API_TOKEN` for the intended deployment.

## Supply chain

- Multi-stage Docker build.
- Minimal runtime image (Alpine + binary).
- CI runs `go vet` and `go test -race`.
