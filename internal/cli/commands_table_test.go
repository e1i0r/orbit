package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

func TestCommandsTableIntegrity(t *testing.T) {
	p := words.For("en")

	for _, cmd := range commands() {
		if cmd.Name == "" {
			t.Error("command table contains command with empty Name")
		}

		if cmd.About == nil || cmd.About(p) == "" {
			t.Errorf("command %q has missing or empty About description", cmd.Name)
		}

		if cmd.InWindow == WindowRefuses && (cmd.Because == nil || cmd.Because(p) == "") {
			t.Errorf("command %q is WindowRefuses but has empty Because description", cmd.Name)
		}

		if cmd.Usage() == "" {
			t.Errorf("command %q has empty Usage()", cmd.Name)
		}
	}
}

// TestNoCommandIsInTheTableTwice. The table is built out of three files now
// that one of them met the size ceiling, and a name added to the wrong one is
// a name in the list twice: `approve` was, and the window's menu drew it in
// two places while lookup answered from the first.
func TestNoCommandIsInTheTableTwice(t *testing.T) {
	seen := map[string]bool{}

	for _, cmd := range commands() {
		if seen[cmd.Name] {
			t.Errorf("%q is in the command table twice", cmd.Name)
		}

		seen[cmd.Name] = true
	}
}

// TestTheVerbsTheWindowReachesOnlyByNameAreStillCalledThat. approve, permit
// and critical have no key in the window: its task menu names them, so a
// rename here takes them out of the window without a compiler saying so.
func TestTheVerbsTheWindowReachesOnlyByNameAreStillCalledThat(t *testing.T) {
	for _, name := range []string{"approve", "permit", "critical"} {
		c, ok := lookup(name)
		if !ok {
			t.Errorf("%q is not in the command table, and the window's task menu asks for it by that name", name)
			continue
		}

		if !c.AboutATask {
			t.Errorf("%q is no longer about one task, so the task menu is the wrong place for it", name)
		}
	}
}

func TestNoteSubcommandExecution(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	// 1. Create a task with -id
	code, out, errOut := run(t, "new", "-repo", repoDir, "-id", "PAY-100", "Improve retry backoff")
	if code != 0 {
		t.Fatalf("new failed (exit %d): out=%q err=%q", code, out, errOut)
	}

	// 2. Add note
	code, out, errOut = run(t, "note", "-repo", repoDir, "PAY-100", "Use exponential backoff with jitter")
	if code != 0 {
		t.Fatalf("note failed (exit %d): out=%q err=%q", code, out, errOut)
	}

	if !strings.Contains(out, "note recorded for PAY-100") {
		t.Errorf("unexpected note output: %q", out)
	}

	// 3. Error cases for note
	if c, _, _ := run(t, "note"); c == 0 {
		t.Error("expected note without args to fail")
	}

	if c, _, _ := run(t, "note", "-repo", repoDir, "PAY-100"); c == 0 {
		t.Error("expected note without text to fail")
	}
}

func TestReposAndReconcileSubcommands(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	// 1. orbit repos with root dir
	code, out, errOut := run(t, "repos", root)
	if code != 0 {
		t.Fatalf("repos failed (exit %d): out=%q err=%q", code, out, errOut)
	}

	if !strings.Contains(out, "payments") {
		t.Errorf("expected repos output to list 'payments', got: %q", out)
	}

	// 2. orbit reconcile
	code, out, errOut = run(t, "reconcile", "-repo", repoDir)
	if code != 0 {
		t.Fatalf("reconcile failed (exit %d): out=%q err=%q", code, out, errOut)
	}
}

func TestFlowsSubcommandDetails(t *testing.T) {
	code, out, errOut := run(t, "flows")
	if code != 0 {
		t.Fatalf("flows failed (exit %d): out=%q err=%q", code, out, errOut)
	}

	if !strings.Contains(out, "task") && !strings.Contains(out, "quick") {
		t.Errorf("expected flows output to mention built-in flows, got: %q", out)
	}
}
