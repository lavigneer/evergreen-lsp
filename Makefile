BINARY_NAME:=evergreenlsp

.PHONY: run build test lint format clean setup

build:
	goreleaser build --clean --single-target --snapshot --output dist/$(BINARY_NAME)

clean:
	go clean
	rm -rf dist

setup:
	go mod tidy

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

test:
	go test ./...

