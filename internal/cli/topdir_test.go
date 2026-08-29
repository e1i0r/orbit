package cli

// What `orbit top <dir>` is a window over, and what makes it different from
// `orbit top <another dir>`.
//
// top_test.go asserts the wiring: that a board was read, folded and printed.
// This file asserts the argument. The command ignored it for the whole of
// task 14 — it reached mustBeDirectory and the header string and nothing
// else — and every test that existed passed, because none of them ever built
// two roots and asked the two boards to differ.

import (
	"path/filepath"
	"strings"
	"testing"
)

// oneRepoNoTasks builds a root holding one repository with nothing written
// against it: the state every new reader is in the first time they run this
// command, and the one most easily described as having no repositories at
// all.
func oneRepoNoTasks(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	emptyHome(t)
	initRepo(t, filepath.Join(root, "payments"))

	return root
}

// A repository under the root counts even though nothing has been written
// against it, and that is what makes the third empty state reachable.
//
// A repository gains a directory under repos/ when its first task is
// written, so a count taken from the state root is zero on every fresh
// install whatever the directory holds — and the frame then said "No
// repositories under ~/code" and offered to clone one, which was both untrue
// and useless. The branch that belongs here was already written and could
// not be reached.
func TestARepositoryWithNoTasksIsCountedAndSaysToWriteOne(t *testing.T) {
	root := oneRepoNoTasks(t)

	code, out, errOut := run(t, "top", root, "-once")
	if code != 0 {
		t.Fatalf("top exited %d: %s", code, errOut)
	}

	for _, want := range []string{"1 repository", "no tasks written", "orbit new"} {
		if !strings.Contains(out, want) {
			t.Errorf("the frame does not say %q:\n%s", want, out)
		}
	}

	if strings.Contains(out, "No repositories under") {
		t.Errorf("the frame says there are no repositories, over a root that holds one:\n%s", out)
	}
}

// The assertion the finding needed, and the one nothing here had.
//
// One root cannot show that the directory was used at all: the record does
// not say where a window was pointed, so a board over one root looks the
// same whether the argument reached the reader or was dropped on the floor.
// Two roots over one state root can — the record holds both tasks, and only
// the directory decides which of them is drawn.
func TestTopOnOneDirectoryDoesNotShowAnothers(t *testing.T) {
	emptyHome(t)
	first, second := t.TempDir(), t.TempDir()
	// Two repositories under the first root and not one, because the
	// repository column is dropped when every row would carry the same
	// value: a frame of one repository never draws the name this test is
	// about, and would pass without it.
	for _, r := range []struct{ root, dir, id, text string }{
		{first, "payments", "ACME-1", "retry the webhook on 5xx"},
		{first, "shipping", "ACME-3", "stop quoting saturday delivery"},
		{second, "billing", "ACME-2", "reconcile the ledger nightly"},
	} {
		dir := filepath.Join(r.root, r.dir)
		initRepo(t, dir)

		if code, _, errOut := run(t, "new", "-repo", dir, "-id", r.id, r.text); code != 0 {
			t.Fatalf("new %s exited %d: %s", r.id, code, errOut)
		}
	}

	code, out, errOut := run(t, "top", first, "-once")
	if code != 0 {
		t.Fatalf("top exited %d: %s", code, errOut)
	}

	for _, want := range []string{"payments", "shipping", "ACME-1", "ACME-3", "2 repos"} {
		if !strings.Contains(out, want) {
			t.Errorf("the frame over the first root does not mention %q:\n%s", want, out)
		}
	}

	for _, other := range []string{"billing", "ACME-2"} {
		if strings.Contains(out, other) {
			t.Errorf("the frame over the first root mentions %q, which is under the second:\n%s", other, out)
		}
	}
}
