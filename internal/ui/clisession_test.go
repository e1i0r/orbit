package ui

// Which task a session is opened on.

import (
	"os/exec"
	"testing"

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
