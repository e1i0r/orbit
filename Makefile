export PATH := /usr/local/go/bin:$(HOME)/go/bin:$(PATH)
GO ?= $(shell which go 2>/dev/null || echo /usr/local/go/bin/go)

.PHONY: check fmt vet lint test integration coverage mutate fuzz tidy build install run site demo tapes posters

# check is what a contributor runs before pushing, so it has to be what CI
# runs: lint used to be in CI and not here, which meant a green local check
# and a red pull request over a rule the contributor never saw.
check: fmt vet lint test integration tidy

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

# integration walks the flows the landing shows, end to end: the real binary,
# a real git repository with a real Go module in it, real gates, and a
# stand-in engine on PATH under the name the flows ask for. Only the model is
# faked, because a model is neither free nor the same twice.
#
# Behind a build tag so that `go test ./...` stays a second's work — these
# build binaries and run git, and a contributor running the unit tests should
# not pay for that. check calls this, so a release cannot go out on flows
# nobody walked.
integration:
	$(GO) test -tags integration -count=1 ./test/integration/...
	$(GO) test -tags integration -count=1 -run TestCockpit ./internal/cli/

# coverage is a gate and not a report: it fails under the floor. A number
# printed and ignored is a number that drifts, and the day somebody notices is
# the day it is 60% and nobody knows which change spent it.
COVERAGE_FLOOR ?= 90

coverage:
	@mkdir -p .coverage
	@$(GO) test -count=1 -coverprofile=.coverage/coverage.out -covermode=atomic ./...
	@total=$$($(GO) tool cover -func=.coverage/coverage.out | tail -n 1 | grep -oE "[0-9]+\.[0-9]+"); \
	echo ""; \
	echo "total statement coverage: $$total% (floor $(COVERAGE_FLOOR)%)"; \
	awk -v t="$$total" -v f="$(COVERAGE_FLOOR)" \
		'BEGIN { if (t+0 < f+0) { printf "coverage is under the floor by %.1f points\n", f-t; exit 1 } }'

# mutate asks whether the tests would notice if the code were wrong. It is not
# part of check because a mutation run takes minutes per package: it is run on
# what a change touched, with PKG=./internal/whatever.
#
# Coverage says a line ran. This says the line matters — a mutant that lives
# is a statement no test disagrees with, which is a test that watches without
# looking.
mutate:
	@command -v gremlins >/dev/null || { 		echo "gremlins is not installed:"; 		echo "  go install github.com/go-gremlins/gremlins/cmd/gremlins@latest"; 		exit 1; 	}
	gremlins unleash --tags "" $(or $(PKG),./internal/ui/settings/...)

# fuzz runs every fuzz target in one package for a while. New corpus entries
# it finds are committed: a crash found once is a case the suite keeps.
fuzz:
	$(GO) test $(or $(PKG),./internal/engine/...) -run=Fuzz -fuzz=Fuzz -fuzztime=$(or $(FOR),60s)

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

# A leading ~ is expanded here and not by the shell. Every path below is
# quoted, because a directory with a space in it has to be, and a quoted ~ is
# not a home directory — it is a directory called "~". `make install
# PREFIX=~/.local` made one inside the checkout and reported success, while
# the binary on PATH stayed the one from before.
DESTBIN := $(patsubst ~%,$(HOME)%,$(BINDIR))

install: build
	@mkdir -p "$(DESTBIN)"
	install -m 0755 orbit "$(DESTBIN)/orbit"
	@echo "installed $(DESTBIN)/orbit"

# run opens the cockpit over the current directory. ARGS is the escape hatch
# for everything top takes — a different root, or none of this and a flag
# instead — so this stays the one command a contributor needs to remember.
run:
	$(GO) run ./cmd/orbit top $(ARGS)

# site writes the landing page, once per language, from web/page.tmpl.html and
# the two catalogues beside it. The pages it writes are committed: GitHub Pages
# serves site/ as it finds it, so nothing runs this at deploy time — which is
# why `make check` fails when what is committed is not what the template says.
site:
	$(GO) run ./web/build
	@echo "site/index.html and site/es/index.html are up to date"

# demo seeds a board to shoot the recordings against: two repositories under
# ~/code and a state root of its own under .demo/, with the record written
# through the doors that write it — orbit new, the migration that runs before
# every command, and orbit supervisor -by. It calls no engine, and it never
# touches the operator's own ~/.orbit.
# The binary is built without the version stamp on purpose: a build that knows
# its own version asks GitHub for a newer one and puts an upgrade banner across
# the header, which would be in every frame of every recording. A build from
# source calls itself dev and is never offered one.
demo:
	$(GO) build -o orbit ./cmd/orbit
	ORBIT_BIN=$(PWD)/orbit assets/tapes/seed.sh $(PWD)/.demo

# tapes shoots every recording that can be made against that board and writes
# assets/ and site/. Three of the nine need a run actually going and are shot
# by hand; assets/tapes/README.md says which and how.
SEEDED = flow-start flow-menus flow-reading flow-supervisor flow-knowledge flow-flows

tapes: demo
	@for t in $(SEEDED); do \
		echo "shooting $$t"; \
		ORBIT_DEMO_HOME=$(PWD)/.demo/home \
		ORBIT_DEMO_FRESH=$(HOME)/.orbit-demo \
		ORBIT_DEMO_BIN=$(PWD) \
		vhs assets/tapes/$$t.tape || exit 1; \
	done
	@$(MAKE) posters

# posters is the still the landing shows before a video is played. It is taken
# from a third of the way in rather than from the first frame: the first frame
# of every one of these is an empty shell, and a page of nine empty shells says
# nothing about what is in them.
#
# A take whose subject arrives late gets its own fraction: flow-start spends
# its first half in a terminal, and the cockpit — which is what the section is
# about — only opens at the end.
posters:
	@for f in site/flow-*.mp4; do \
		out=$${f%.mp4}-poster.jpg; \
		case "$$f" in *flow-start.mp4) frac=0.82 ;; *) frac=0.33 ;; esac; \
		dur=$$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$$f"); \
		at=$$(awk -v d="$$dur" -v r="$$frac" 'BEGIN { printf "%.2f", d * r }'); \
		ffmpeg -y -v error -ss "$$at" -i "$$f" -frames:v 1 -q:v 3 "$$out"; \
		echo "poster $$out at $${at}s"; \
	done
