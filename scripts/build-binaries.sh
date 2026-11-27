#!/bin/sh
set -eu
output=${1:?output directory required}
suffix=${2:-}
mkdir -p "$output"
export CGO_ENABLED=0
go build -trimpath -o "$output/agent-factory$suffix" ./cmd/goal
go build -trimpath -o "$output/factory$suffix" ./cmd/factory
go build -trimpath -o "$output/api$suffix" ./cmd/api
