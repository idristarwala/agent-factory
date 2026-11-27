# Deploy the production API

Use the Helm chart to run the HTTP API against a shared PostgreSQL database. The API evaluates submitted workspace runs in its own process. Before an upgrade, let active runs reach review or completion. Production data and revision history reside in PostgreSQL. [Workspace behavior](API.md) · [Storage and configuration](CONFIGURATION.md)

The engine serves the compiled operator UI at the root of the API service. Route both the workspace and its API through the same origin. [Operator workspace](UI.md)

## Prepare a release

1. Select a tested engine release image. Copy its repository and digest from the release manifest or from your registry after building and publishing the image. Keep the manifest with the deployment revision. [Release artifacts](RELEASE.md)
2. Provision PostgreSQL and a database account that can create the runtime tables. Configure a database backup and verify restoration in a separate database before the first upgrade.
3. In the target namespace, provision a Kubernetes Secret named `factory-runtime` containing `database-url`, `gemini-api-key` and `api-token`. Use your deployment system’s secret integration. Choose a different model provider by configuring its documented variables. [Authentication](../AUTH.md)
4. Save the following Helm values in your deployment repository. Replace the image digest placeholder with the full verified digest.

```yaml
image:
  repository: ghcr.io/idristarwala/agent-factory-starter
  digest: "sha256:<64 lowercase hex characters>"
env:
  FACTORY_STORE_MODE: postgres
  FACTORY_ID: production
  LLM_PROVIDER: gemini
secretEnv:
  DATABASE_URL: {name: factory-runtime, key: database-url}
  GEMINI_API_KEY: {name: factory-runtime, key: gemini-api-key}
  API_TOKEN: {name: factory-runtime, key: api-token}
secrets:
  enabled: false
```

For a mounted gateway or tool configuration, set `secrets.enabled: true`, `secrets.name` to your existing file Secret and `secrets.optional: false`. Set `env.SECRETS_DIR` to the desired mount directory and point the corresponding configuration variables at files there. Environment values sourced from a Secret are read at process startup. Roll the deployment after rotating those values.

## Roll out and verify

From a checkout of the selected chart revision, run:

```sh
chart=deploy/helm/agent-factory
helm lint "$chart" -f values-production.yaml
helm upgrade --install factory "$chart" --namespace factory --create-namespace \
  --values values-production.yaml --atomic --wait --timeout 5m
kubectl -n factory rollout status deployment/factory-agent-factory
kubectl -n factory port-forward service/factory-agent-factory 18080:8080
```

From another terminal run `curl -fsS http://localhost:18080/health`. Expect `status` to be `ok` and `store` to be `postgres`. Use the configured bearer token for `GET /api/settings`, then submit a representative record and review its decision through the API. Readiness checks API availability. A representative run also exercises model and tool credentials. The Service defaults to ClusterIP. Configure your cluster’s ingress and TLS when exposing it externally. [API](API.md) · [Operations](OPERATIONS.md)

Use one release per namespace with this chart’s existing Deployment selector. `service.port` selects the client-facing port and `containerPort` selects the API listener. The Service forwards to the named HTTP port.

## Roll back

Use `helm history factory -n factory` to identify a known-good revision. Check that the previous engine can read the current database schema before running `helm rollback factory REVISION -n factory --wait`. Repeat the health and representative-run checks. A Helm rollback restores Kubernetes resources. PostgreSQL recovery uses your database backup procedure.
