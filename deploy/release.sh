#!/bin/sh
set -eu
for target in linux/amd64 darwin/amd64 darwin/arm64 windows/amd64; do
  export GOOS="${target%/*}" GOARCH="${target#*/}"
  suffix="-$GOOS-$GOARCH"
  if [ "$GOOS" = windows ]; then suffix="$suffix.exe"; fi
  scripts/build-binaries.sh dist "$suffix"
done
