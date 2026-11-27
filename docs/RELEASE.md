# Releases

Run `make release` to build the goal runner (`agent-factory`), management CLI (`factory`), and HTTP server (`api`) in `dist/`. Published targets are Linux amd64, macOS amd64 and arm64, and Windows amd64. The container builds the same three programs for its target platform.

A version tag triggers the release workflow. It builds the binaries, publishes the container, records its digest and binary filenames in the release manifest, and attaches the artifacts to the GitHub release. Ordinary pull requests run vet, race tests, the Go build, and the frontend build.
