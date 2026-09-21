package repo

// What a checkout already refuses work over.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkedIn writes one workflow into a scratch checkout.
func checkedIn(t *testing.T, name, body string) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, ".github", "workflows", name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	return root
}

// TestWhatAPullRequestHasToPassIsWhatThisBrings.
//
// The whole of why this source is worth its own reading: the command exists
// already and is already refusing work, so a rule out of it arrives with a
// gate nobody had to invent.
func TestWhatAPullRequestHasToPassIsWhatThisBrings(t *testing.T) {
	root := checkedIn(t, "check.yml", `
name: check
on:
  pull_request:
jobs:
  check:
    steps:
      - uses: actions/checkout@v5
      - run: go vet ./...
      - uses: golangci/golangci-lint-action@v9
      - run: make test
`)

	got := Gates(root)
	if len(got) != 3 {
		t.Fatalf("read %d gates out of one workflow: %+v", len(got), got)
	}

	for i, want := range []string{"go vet ./...", "golangci-lint run", "make test"} {
		if got[i].Command != want {
			t.Errorf("gate %d is %q, want %q", i, got[i].Command, want)
		}

		if !strings.HasSuffix(got[i].Where, "check.yml") {
			t.Errorf("gate %d does not say which file it came out of: %q", i, got[i].Where)
		}
	}
}

// TestAWorkflowNothingHasToPassBringsNothing.
//
// A release builds what was already agreed to and a schedule runs on nobody's
// change. Neither stands between a change and the branch, so a rule taken out
// of one would refuse work over a command that has never refused any.
func TestAWorkflowNothingHasToPassBringsNothing(t *testing.T) {
	root := checkedIn(t, "release.yml", `
name: release
on:
  workflow_dispatch:
  release:
    types: [published]
jobs:
  ship:
    steps:
      - run: goreleaser release
`)

	if got := Gates(root); len(got) != 0 {
		t.Errorf("a workflow no change goes through brought %+v", got)
	}
}

// TestAScriptIsNotACommand.
//
// A block opened with | is somebody's shell program — several commands, a
// conditional, a heredoc. Handing one to a gate is handing it something
// nobody wrote to be a gate, and whoever kept it finds out on the next run.
func TestAScriptIsNotACommand(t *testing.T) {
	root := checkedIn(t, "check.yml", `
on: [pull_request]
jobs:
  check:
    steps:
      - run: |
          set -e
          go build ./...
          go test ./...
      - run: go vet ./...
`)

	got := Gates(root)
	if len(got) != 1 || got[0].Command != "go vet ./..." {
		t.Errorf("a script was read as a command: %+v", got)
	}
}

// TestOneCommandIsOfferedOnce, however many workflows run it: a tray with
// the same gate in it twice is a tray asking the same question twice.
func TestOneCommandIsOfferedOnce(t *testing.T) {
	root := checkedIn(t, "a.yml", "on: [pull_request]\njobs:\n  a:\n    steps:\n      - run: make check\n")
	if err := os.WriteFile(filepath.Join(root, ".github", "workflows", "b.yml"),
		[]byte("on:\n  push:\njobs:\n  b:\n    steps:\n      - run: make check\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := Gates(root); len(got) != 1 {
		t.Errorf("one command read twice was offered %d times: %+v", len(got), got)
	}
}

// TestACheckoutWithNoWorkflowsSaysSo, rather than answering with something
// it did not read.
func TestACheckoutWithNoWorkflowsSaysSo(t *testing.T) {
	if got := Gates(t.TempDir()); len(got) != 0 {
		t.Errorf("a checkout with no workflows brought %+v", got)
	}
}

// TestNoMoreThanAHandfulAreOffered. They share one tray with three other
// sources, and a tray of forty weak offers is a tray somebody stops opening.
func TestNoMoreThanAHandfulAreOffered(t *testing.T) {
	var b strings.Builder

	b.WriteString("on: [pull_request]\njobs:\n  check:\n    steps:\n")

	for i := range 20 {
		b.WriteString("      - run: step-" + string(rune('a'+i)) + "\n")
	}

	if got := Gates(checkedIn(t, "check.yml", b.String())); len(got) != atMostGates {
		t.Errorf("twenty steps offered %d gates, want the cap of %d", len(got), atMostGates)
	}
}

// TestAHookIsAGate. It runs on every commit on the machine of whoever
// installed it, which is a rule the project is already keeping.
func TestAHookIsAGate(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".pre-commit-config.yaml"),
		[]byte("repos: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Gates(root)
	if len(got) != 1 || got[0].Command != "pre-commit run --all-files" {
		t.Errorf("a pre-commit config brought %+v", got)
	}
}

// TestALinterConfigIsAGate. A project does not carry a .golangci.yml by
// accident: somebody configured that tool for this code, and the rule is
// that the code passes it.
func TestALinterConfigIsAGate(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".golangci.yml"), []byte("linters:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Gates(root)
	if len(got) != 1 || got[0].Command != "golangci-lint run" {
		t.Errorf("a linter config brought %+v", got)
	}

	if got[0].Where != ".golangci.yml" {
		t.Errorf("it does not say which file said so: %q", got[0].Where)
	}
}

// TestTheStrongestClaimWins.
//
// A workflow running a linter and that linter's own config are the same
// command twice. The workflow is the one to keep — it is the reading that
// says something actually refuses work — and the config then adds nothing.
func TestTheStrongestClaimWins(t *testing.T) {
	root := checkedIn(t, "check.yml", `
on: [pull_request]
jobs:
  check:
    steps:
      - uses: golangci/golangci-lint-action@v9
`)

	if err := os.WriteFile(filepath.Join(root, ".golangci.yml"), []byte("linters:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Gates(root)
	if len(got) != 1 {
		t.Fatalf("one command read twice was offered %d times: %+v", len(got), got)
	}

	if got[0].Where != ".github/workflows/check.yml" {
		t.Errorf("the weaker reading won: %q", got[0].Where)
	}
}

// TestALocalHookIsNotAGate.
//
// .git/hooks does not travel. A rule written inside the repository about a
// hook only this machine has is a rule that refuses work for everybody who
// clones the project and has nothing to run.
func TestALocalHookIsNotAGate(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, ".git", "hooks", "pre-commit"),
		[]byte("#!/bin/sh\nmake check\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := Gates(root); len(got) != 0 {
		t.Errorf("a hook that does not travel brought %+v", got)
	}
}

// TestTheCommandAGateRunsIsTheOneTheWorkflowWrote.
//
// A workflow's `run:` is YAML, so a command with a colon in it arrives
// quoted and the quotes are not part of the command. A pair that is not a
// pair is not quoting: `'make check` is somebody's typo and handing a gate
// the text without its apostrophe would run something they did not write.
//
// And a block opened with | or > is a shell program — several commands, a
// conditional, a heredoc — which is not a gate whatever it says.
func TestTheCommandAGateRunsIsTheOneTheWorkflowWrote(t *testing.T) {
	for _, one := range []struct {
		why  string
		run  string
		want string
	}{
		{"an unquoted command is itself", "make check", "make check"},
		{"single quotes are the YAML and not the command", "'go test ./...'", "go test ./..."},
		{"and so are double ones", `"go test ./..."`, "go test ./..."},
		{"quotes that are not a pair are not quoting", "'make check", "'make check"},
		{"nor are two different ones", `'make check"`, `'make check"`},
		{"one character cannot be a pair", "'", "'"},
		{"a pair around nothing is nothing", "''", ""},
		{"a block is not a command", "|", ""},
		{"nor is one with a program under it", "|\n  make check\n  make lint", ""},
		{"nor a folded one", ">", ""},
		{"and the space around it is not part of it", "  make check  ", "make check"},
	} {
		if got := oneCommand(one.run); got != one.want {
			t.Errorf("oneCommand(%q) = %q, want %q — %s", one.run, got, one.want, one.why)
		}
	}
}
