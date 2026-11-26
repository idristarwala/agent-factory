#!/usr/bin/env bash
set -euo pipefail
echo "Setting up Agent Factory..."
mkdir -p secrets
if [ -n "${GEMINI_API_KEY:-}" ]; then
  echo -n "$GEMINI_API_KEY" > secrets/gemini-api-key
  echo "Wrote secrets/gemini-api-key"
fi
go mod download
echo "Ready. Run: go run ./cmd/goal \"your goal\""
