.PHONY: test build ci release

test:
	go test -race ./...

build:
	go build ./...

ci: test build
	go vet ./...

release:
	@echo "Build binaries:"
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/agent-factory-linux-amd64 ./cmd/goal
	GOOS=darwin GOARCH=amd64 go build -o dist/agent-factory-darwin-amd64 ./cmd/goal
	GOOS=windows GOARCH=amd64 go build -o dist/agent-factory-windows-amd64.exe ./cmd/goal
	@echo "Done. See dist/"
