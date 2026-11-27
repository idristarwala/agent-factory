.PHONY: test build ci ui-build release

test:
	go test -race ./...

build:
	go build ./...

ui-build:
	npm --prefix ui-app ci
	npm --prefix ui-app test
	npm --prefix ui-app run build

ci: test build ui-build conformance docs-check
	go vet ./...

release:
	./deploy/release.sh

.PHONY: test-gemini
test-gemini:
	go test -tags gemini_live -run '^TestLiveGemini' -count=1 ./internal/factory ./internal/llm

.PHONY: conformance
conformance:
	go test -race ./internal/llm/... ./internal/configuredgateway
	cd examples/out-of-tree-gateway && go test -race ./...

.PHONY: test-mixed-judges
test-mixed-judges:
	go test -v -tags openrouter_live -run '^TestLiveOpenRouter' -count=1 ./internal/factory ./internal/llm

.PHONY: docs-check
docs-check:
	python3 scripts/check-docs.py
