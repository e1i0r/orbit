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

// wayInTheWindow is how each verb about a task is reached without typing it:
// everything the command line can do to a task, the window can do from a
// menu. The commands about no task in particular need no entry here — the
// board's menu is built from the table itself, so they are all on it.
//
// The window's task menu asks for most of these by name, and a rename here
// takes one out of the window with nothing to say so; a new verb about a
// task with no line in this map is one nobody has given a way in yet.
var wayInTheWindow = map[string]string{
	"pause":    "the pause key and the task's menu, through the control port",
	"resume":   "the resume key and the task's menu, through the control port",
	"skip":     "the skip key and the task's menu, through the control port",
	"cancel":   "the cancel key and the task's menu, through the control port",
	"requeue":  "the requeue key and the task's menu, through the control port",
	"read":     "the mark-read key and the task's menu, through its own port",
	"show":     "opening the task, which is what the menu's first entry does",
	"run":      "the start dialog, from its key or the task's menu",
	"note":     "the message box, from its key or the task's menu",
	"direct":   "the message box, from the task's menu",
	"pr":       "the task's menu; the deliver key asks the supervisor instead",
	"merge":    "the task's menu, and the merge key on the task's screen",
	"close-pr": "the task's menu, and the close key on the task's screen",
	"resolve":  "the task's menu; the deliver key asks the supervisor instead",
	"approve":  "the task's menu",
	"permit":   "the task's menu",
	"critical": "the task's menu",
}

func TestEveryVerbAboutATaskHasAWayInTheWindow(t *testing.T) {
	for _, c := range commands() {
		if c.AboutATask && wayInTheWindow[c.Name] == "" {
			t.Errorf("%q is about a task and has no way in the window; the command line is the only place it can be run from", c.Name)
		}
	}

	for name := range wayInTheWindow {
		c, ok := lookup(name)
		if !ok {
			t.Errorf("%q is not in the command table, and the window asks for it by that name", name)
			continue
		}

		if !c.AboutATask {
			t.Errorf("%q is no longer about one task, so the task's menu is the wrong place for it", name)
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

// TestTheFlagTerminatorIsNotPartOfWhatWasSaid. note and direct take their
// message after the id, which is where flag parsing has already stopped: a
// "--" there is not a terminator the flag package removes, it is the first
// word of the message. The window put one in front of every note it wrote,
// and every note it wrote started with it.
func TestTheFlagTerminatorIsNotPartOfWhatWasSaid(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	if code, out, errOut := run(t, "new", "-repo", repoDir, "-id", "PAY-200", "Retry backoff"); code != 0 {
		t.Fatalf("new failed (exit %d): out=%q err=%q", code, out, errOut)
	}

	if code, out, errOut := run(t, "note", "-repo", repoDir, "PAY-200", "--", "use jitter"); code != 0 {
		t.Fatalf("note failed (exit %d): out=%q err=%q", code, out, errOut)
	}

	if code, out, errOut := run(t, "direct", "-repo", repoDir, "PAY-200", "--", "stop and ask"); code != 0 {
		t.Fatalf("direct failed (exit %d): out=%q err=%q", code, out, errOut)
	}

	code, out, errOut := run(t, "show", "-repo", repoDir, "PAY-200")
	if code != 0 {
		t.Fatalf("show failed (exit %d): out=%q err=%q", code, out, errOut)
	}

	if strings.Contains(out, "--") {
		t.Errorf("what the task was told keeps the flag terminator: %q", out)
	}

	for _, said := range []string{"use jitter", "stop and ask"} {
		if !strings.Contains(out, said) {
			t.Errorf("the task does not carry %q: %q", said, out)
		}
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
