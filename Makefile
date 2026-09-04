export PATH := /usr/local/go/bin:$(HOME)/go/bin:$(PATH)
GO ?= $(shell which go 2>/dev/null || echo /usr/local/go/bin/go)

.PHONY: check fmt vet lint test coverage tidy build install run

# check is what a contributor runs before pushing, so it has to be what CI
# runs: lint used to be in CI and not here, which meant a green local check
# and a red pull request over a rule the contributor never saw.
check: fmt vet lint test tidy

fmt:
	@test -z "$$($(GO) fmt ./...)" || { echo "gofmt made changes — commit them"; exit 1; }

# Twice, because a build constraint hides code from whichever platform is not
# running: internal/task has three bootTime implementations behind //go:build,
# and a mac compiles exactly one of them. CI runs on Linux, so the pass below
# is the one that makes a green check here mean a green check there.
vet:
	$(GO) vet ./...
	GOOS=linux $(GO) vet ./...

# The Linux pass is here for the reason it is on vet, and it is where the
# blank-line rules were first felt: the fixer never saw boot_linux.go.
#
# A missing golangci-lint skips, loudly, rather than failing: a contributor
# without the binary still gets the whole of the rest of check, and CI — where
# the binary is always there — still enforces it.
lint:
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run ./...; \
		GOOS=linux golangci-lint run ./...; \
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

# VERSION is what the build calls itself, and it is asked of git rather than
# written down here: a number kept in the file goes stale the moment somebody
# forgets to bump it. `git describe` answers with the last release tag, how
# far past it this checkout is, and -dirty when the tree has uncommitted work
# — which is the whole question a reader has when the window shows a version
# and they are wondering whether their fix is in the binary they are running.
#
# A checkout with no tags at all, and a source tree that is not a checkout,
# both fall back to the "dev" the code already defaults to.
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS := -X github.com/e1i0r/orbit/internal/cli.Version=$(VERSION)

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o orbit ./cmd/orbit

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

# run opens the cockpit over the current directory. ARGS is the escape hatch
# for everything top takes — a different root, or none of this and a flag
# instead — so this stays the one command a contributor needs to remember.
run:
	$(GO) run ./cmd/orbit top $(ARGS)
