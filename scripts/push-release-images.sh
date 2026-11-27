#!/bin/sh
# Emit the pushed registry digests to the GitHub Actions environment.
set -eu
: "${IMAGE_REPOSITORY:?}" "${GITHUB_REF_NAME:?}" "${GITHUB_ENV:?}"
metadata=$(mktemp)
trap 'rm -f "$metadata"' EXIT
for role in engine worker connector; do
  tag=$GITHUB_REF_NAME
  if [ "$role" != engine ]; then tag="$tag-$role"; fi
  docker buildx build --target "$role" --tag "$IMAGE_REPOSITORY:$tag" --push --metadata-file "$metadata" .
  digest=$(jq -er '."containerimage.digest" | select(test("^sha256:[0-9a-f]{64}$"))' "$metadata")
  key=$(echo "$role" | tr '[:lower:]' '[:upper:]')
  echo "${key}_DIGEST=$digest" >> "$GITHUB_ENV"
done
