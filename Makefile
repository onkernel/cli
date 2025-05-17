build:
	go build -o bin/kernel ./cmd/kernel

test:
	go test ./...

lint:
	@golangci-lint run || true

changelog:
	@chglog --next-tag $(shell git describe --tags --abbrev=0)

release-dry-run:
	goreleaser check
	goreleaser healthcheck
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean

.PHONY: build test lint run changelog release 
