# Working with Agent Factory

To drive a factory from another agent or framework, start with the [harness quick start](docs/HARNESSES.md). The CLI accepts JSON files and emits one JSON object per evaluated record. The HTTP API supports shared sessions, versioned definitions and human review. [API reference](docs/API.md)

For repository changes, identify the tracking issue, verify the affected runtime entry point and update its operator guide. Keep framework code independent of example domains. Preserve immutable definition and rule history for both human and steward edits. Run `make ci` for Go, UI and gateway checks. Live model tests additionally require provider credentials. [Configuration](docs/CONFIGURATION.md) · [Project status](docs/STATUS.md)
