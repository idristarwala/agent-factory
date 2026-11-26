#!/usr/bin/env bash
set -euo pipefail
mkdir -p secrets
if [ -n "${GEMINI_API_KEY:-}" ] && [ ! -f secrets/gemini-api-key ]; then
  echo -n "$GEMINI_API_KEY" > secrets/gemini-api-key
fi
docker compose up --build
