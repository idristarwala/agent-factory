.PHONY: test build ci ui-build release

test:
	go test -race ./...

build:
	go build ./...

ui-build:
	npm --prefix ui-app ci
	npm --prefix ui-app run build

ci: test build ui-build
	go vet ./...

release:
	./deploy/release.sh
