#!/usr/bin/env bash
set -euo pipefail
GOAL="${1:-Fix catalog titles}"
export SECRETS_DIR="${SECRETS_DIR:-$(pwd)/secrets}"
export LLM_PROVIDER="${LLM_PROVIDER:-gemini}"
go run ./cmd/goal "$GOAL"
