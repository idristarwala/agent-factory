#!/bin/bash
# Release pipeline for issue #9.
# Builds binaries and emits a sample release manifest.
# Full CI automation of docker build + inspect remains the next step.

set -e

VERSION="${1:-v0.1.0}"

echo "Building factory binaries for ${VERSION}..."
mkdir -p bin dist
go build -o bin/factory ./cmd/factory
go build -o bin/continuous-swarm ./cmd/continuous-swarm

GOOS=linux GOARCH=amd64 go build -o dist/agent-factory-linux-amd64 ./cmd/factory

# Sample manifest (real digests come from docker inspect after push)
MANIFEST="dist/release-manifest-${VERSION}.json"
cat > "${MANIFEST}" << MANIFEST_EOF
{
  "version": "${VERSION}",
  "engine_digest": "sha256:replace-after-docker-push",
  "worker_digest": "sha256:replace-after-docker-push",
  "pack_spec_version": "1",
  "schema_version": "1",
  "binaries": [
    "dist/agent-factory-linux-amd64"
  ],
  "notes": "Pin this digest in factory.yaml. See docs/RELEASE.md for the docker build + inspect steps."
}
MANIFEST_EOF

echo ""
echo "Sample release manifest written to ${MANIFEST}"
echo "To complete a digest-pinned release:"
echo "1. docker build -t agent-factory:${VERSION} ."
echo "2. docker push agent-factory:${VERSION}"
echo "3. DIGEST=\$(docker inspect --format='{{index .RepoDigests 0}}' agent-factory:${VERSION})"
echo "4. Update the manifest engine_digest and commit it with the tag."
echo ""
echo "Release skeleton complete for ${VERSION}."
