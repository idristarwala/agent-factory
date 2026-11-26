#!/bin/bash
# Release pipeline skeleton for issue #9.
# Builds binaries and prints digest-pinning instructions.

set -e

echo "Building factory binaries..."
go build -o bin/factory ./cmd/factory
go build -o bin/continuous-swarm ./cmd/continuous-swarm

echo ""
echo "To create a digest-pinned release:"
echo "1. docker build -t agent-factory:v0.1.0 ."
echo "2. docker push agent-factory:v0.1.0"
echo "3. docker inspect --format='{{index .RepoDigests 0}}' agent-factory:v0.1.0"
echo "4. Record the digest in the release manifest"
echo ""
echo "Release skeleton complete."
