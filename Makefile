export PATH := /usr/local/go/bin:$(HOME)/go/bin:$(PATH)
GO ?= $(shell which go 2>/dev/null || echo /usr/local/go/bin/go)

.PHONY: check fmt vet lint test coverage tidy build install

# check is what a contributor runs before pushing, so it has to be what CI
# runs: lint used to be in CI and not here, which meant a green local check
# and a red pull request over a rule the contributor never saw.
check: fmt vet lint test tidy

fmt:
	@test -z "$$($(GO) fmt ./...)" || { echo "gofmt made changes — commit them"; exit 1; }

vet:
	$(GO) vet ./...

# A missing golangci-lint skips, loudly, rather than failing: a contributor
# without the binary still gets the whole of the rest of check, and CI — where
# the binary is always there — still enforces it.
lint:
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint is not installed — skipping lint; CI runs it (go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest)"; \
	fi

test:
	$(GO) test ./...

coverage:
	@mkdir -p .coverage
	@$(GO) test -count=1 -coverprofile=.coverage/coverage.out -covermode=atomic ./...
	@echo ""
	@echo "🎯 Total Project Statement Coverage:"
	@$(GO) tool cover -func=.coverage/coverage.out | tail -n 1

# go.mod is tidy or arch.approved's guarantee about indirect requires does
# not hold: an untidy go.mod can carry a module that no import justifies.
tidy:
	$(GO) mod tidy -diff

build:
	$(GO) build -o orbit ./cmd/orbit

# install puts the binary where the shell will find it. PREFIX is honoured so
# a packager, or somebody without write access to /usr/local, can redirect it
# without editing this file; the default is the one directory a Homebrew mac
# and a plain Linux both already have on PATH.
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

install: build
	@mkdir -p "$(BINDIR)"
	install -m 0755 orbit "$(BINDIR)/orbit"
	@echo "installed $(BINDIR)/orbit"
