# ypcli — Makefile
# All targets use the local Go toolchain with CGO disabled.

BINARY      := ypcli
MODULE      := github.com/dantte-lp/ypcli
PKG         := ./cmd/ypcli

VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS     := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

export CGO_ENABLED := 0

.DEFAULT_GOAL := help

## help: Show this help
.PHONY: help
help:
	@grep -E '^## [a-zA-Z_-]+:' $(MAKEFILE_LIST) | sed 's/## //' | awk -F ':' '{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## build: Build the ypcli binary
.PHONY: build
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

## install: Install ypcli into GOBIN
.PHONY: install
install:
	go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)

## test: Run tests with the race detector and coverage
.PHONY: test
test:
	CGO_ENABLED=1 go test -race -coverprofile=coverage.out -covermode=atomic ./...

## cover: Show coverage summary
.PHONY: cover
cover: test
	go tool cover -func=coverage.out | tail -1

## lint: Run golangci-lint
.PHONY: lint
lint:
	golangci-lint run

## lint-docs: Lint Markdown, YAML and spelling
.PHONY: lint-docs
lint-docs:
	npx --yes markdownlint-cli2 "**/*.md" || true
	yamllint . || true
	npx --yes cspell "**/*.md" || true

## fmt: Format Go sources
.PHONY: fmt
fmt:
	gofmt -w .
	go run golang.org/x/tools/cmd/goimports@latest -w .

## tidy: Tidy go.mod
.PHONY: tidy
tidy:
	go mod tidy

## vuln: Run govulncheck
.PHONY: vuln
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## verify: build + test + lint + vuln
.PHONY: verify
verify: build test lint vuln

## snapshot: Build a local goreleaser snapshot
.PHONY: snapshot
snapshot:
	goreleaser release --snapshot --clean

## release-check: Validate the goreleaser config
.PHONY: release-check
release-check:
	goreleaser check

## clean: Remove build artifacts
.PHONY: clean
clean:
	rm -rf $(BINARY) $(BINARY).exe dist coverage.out coverage.html
