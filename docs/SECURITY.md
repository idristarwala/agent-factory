# Security

## Secrets

- Never commit secrets.
- Use the centralized loader (`SECRETS_DIR`).
- Mount secrets read-only in Docker and Kubernetes.
- Rotate by replacing the file; the loader re-reads on mtime change.

## Auth methods

See `AUTH.md`. BigQuery supports ADC, service account, Workload Identity, and impersonation. LLM gateways use API keys or bearer tokens from the secrets directory.

## Judge fail-closed

If the model errors or returns an unclear reply, the proposal is rejected. No silent approvals.

## Network

- Outbound calls only to configured LLM endpoints and tool backends.
- No inbound ports in the default image.

## Supply chain

- Multi-stage Docker build.
- Minimal runtime image (Alpine + binary).
- CI runs `go vet` and `go test -race`.
