#!/usr/bin/env bash
set -euo pipefail
export SECRETS_DIR="${SECRETS_DIR:-$(pwd)/secrets}"
export LLM_PROVIDER="${LLM_PROVIDER:-gemini}"
exec go run ./cmd/goal "$@"
