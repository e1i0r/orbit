package ui

// Which task a session is opened on.

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// TestASessionIsOpenedOnTheTaskTheViewIsOn.
//
// The launcher read the board's cursor, which on the task view is on
// whatever row the reader left it on. The session was told about that task
// and opened in its directory, so c on a task the banner names opened a
// session about another one.
func TestASessionIsOpenedOnTheTaskTheViewIsOn(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2701")
	m.screen, m.detail = screenDetail, "ACME-2662"

	var opened view.Task

	m.opts.Open = func(task view.Task, engineName, dir string) (*exec.Cmd, error) {
		opened = task
		return &exec.Cmd{Path: engineName, Dir: dir}, nil
	}

	after, cmd := advance(t, m, press("c"))
	if cmd == nil {
		t.Fatal("c on the task view answered with no command")
	}

	if opened.ID != "ACME-2662" {
		t.Errorf("the session was opened on %q, want the task the view is on", opened.ID)
	}

	if after.message == "" {
		t.Error("the window said nothing about handing the terminal over")
	}
}

// TestASessionFromTheBoardIsOpenedOnTheRow is the same launcher where there
// is no view: the cursor is the only answer there is.
func TestASessionFromTheBoardIsOpenedOnTheRow(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2701")

	var opened view.Task

	m.opts.Open = func(task view.Task, engineName, dir string) (*exec.Cmd, error) {
		opened = task
		return &exec.Cmd{Path: engineName, Dir: dir}, nil
	}

	if _, cmd := advance(t, m, press("c")); cmd == nil {
		t.Fatal("c on the board answered with no command")
	}

	if opened.ID != "ACME-2701" {
		t.Errorf("the session was opened on %q, want the row under the cursor", opened.ID)
	}
}

// TestComingBackFromATaskSessionAsksNothing.
//
// The question is about writing a task down. A session opened on one is
// already about a task, and answering yes to it wrote a second task for the
// work that was just done in the first.
func TestComingBackFromATaskSessionAsksNothing(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	after, _ := advance(t, m, cliEndedMsg{Engine: "claude", Repo: "/repo", TaskID: "ACME-2662"})
	if after.confirm != confirmNone {
		t.Errorf("confirm = %v after a session on a task, want no question", after.confirm)
	}

	if !strings.Contains(ansi.Strip(after.bandLine(100)), "ACME-2662") {
		t.Errorf("the band says %q, want which session ended", ansi.Strip(after.bandLine(100)))
	}
}

// TestComingBackFromABareSessionStillAsks: opened on no task, what was
// worked out in it has nowhere to go until the reader writes it down.
func TestComingBackFromABareSessionStillAsks(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	after, _ := advance(t, m, cliEndedMsg{Engine: "claude", Repo: "/repo"})
	if after.confirm != confirmPostCliTask {
		t.Errorf("confirm = %v after a session on no task, want the question", after.confirm)
	}
}
