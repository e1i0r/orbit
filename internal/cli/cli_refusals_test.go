package cli

// Every refusal every command owns, asked for three ways: with nothing,
// somewhere that is not a repository, and about a task nobody wrote.
//
// One table rather than thirty tests, because the shape is the same
// everywhere — the id is missing, the directory is not a checkout, the
// task is not in the record — and what is covered is that each command
// refuses before it acts, in its own words, without running anything.

import (
	"os"
	"path/filepath"
	"testing"
)

// refusalArgs is what a command needs past the id to get as far as the
// record: a note with no text is refused for having no text, which is a
// refusal but not the one this table is about.
var refusalArgs = map[string][]string{
	"note":   {"a helpful note"},
	"direct": {"switch to redis"},
}

// TestEveryCommandRefusesBeforeItActs.
func TestEveryCommandRefusesBeforeItActs(t *testing.T) {
	root, orbitHome := workspace(t)
	repoDir := filepath.Join(root, "payments")
	missing := filepath.Join(t.TempDir(), "not-a-checkout")

	var names []string

	for _, c := range commands() {
		if c.AboutATask {
			names = append(names, c.Name)
		}
	}

	if len(names) == 0 {
		t.Fatal("no command is about a task")
	}

	for _, name := range names {
		// With nothing at all.
		if code, _, _ := run(t, name); code == 0 {
			t.Errorf("%s with nothing exited 0", name)
		}

		// Somewhere that is not a repository.
		args := append([]string{name, "-repo", missing, "ACME-1"}, refusalArgs[name]...)
		if code, _, _ := run(t, args...); code == 0 {
			t.Errorf("%s against nowhere exited 0", name)
		}

		// About a task nobody wrote.
		args = append([]string{name, "-repo", repoDir, "ACME-404"}, refusalArgs[name]...)
		if code, _, _ := run(t, args...); code == 0 {
			t.Errorf("%s about a task nobody wrote exited 0", name)
		}
	}

	// A refusal writes nothing down: the record of a workspace nobody
	// acted on is the absence of one.
	if _, err := os.Stat(filepath.Join(orbitHome, "tasks")); err == nil {
		t.Error("refusals left tasks behind")
	}
}
