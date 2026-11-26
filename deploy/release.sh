#!/bin/bash
# Release pipeline for issue #9.
# Builds binaries and documents digest-pinned image steps.
# Full automation (docker build + inspect + manifest commit) is the remaining gap.

set -e

VERSION="${1:-v0.1.0}"

echo "Building factory binaries for ${VERSION}..."
mkdir -p bin dist
go build -o bin/factory ./cmd/factory
go build -o bin/continuous-swarm ./cmd/continuous-swarm

# Cross-compile skeleton for CI
GOOS=linux GOARCH=amd64 go build -o dist/agent-factory-linux-amd64 ./cmd/factory

echo ""
echo "To create a digest-pinned release (manual steps until full CI automation):"
echo "1. docker build -t agent-factory:${VERSION} ."
echo "2. docker push agent-factory:${VERSION}"
echo "3. DIGEST=\$(docker inspect --format='{{index .RepoDigests 0}}' agent-factory:${VERSION})"
echo "4. Record the digest in a release manifest (docs/RELEASE.md or GitHub release)"
echo ""
echo "Release skeleton complete for ${VERSION}."
