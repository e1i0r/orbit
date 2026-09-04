package ui

// Which task a session is opened on.

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

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

	after, _ := advance(t, m, cliEndedMsg{Engine: "claude", Repo: "/repo", Task: view.Task{ID: "ACME-2662"}})
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

// TestTheConversationComesBackWithTheReader.
//
// The window is suspended for the whole session and sees none of it, so
// what was said in it lived only in the engine's own file. Elio asked for
// it to belong to the task it was said on; this is the gesture that puts it
// there, and the port is asked about the session that just ended and not
// about the directory in general.
func TestTheConversationComesBackWithTheReader(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	var (
		asked  view.Task
		engine string
		since  time.Time
	)

	m.opts.FileSession = func(task view.Task, engineName string, from time.Time) (int, error) {
		asked, engine, since = task, engineName, from

		return 4, nil
	}

	started := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)

	_, cmd := advance(t, m, cliEndedMsg{Engine: "claude", Repo: "/repo", Started: started, Task: view.Task{ID: "ACME-2662"}})
	if cmd == nil {
		t.Fatal("coming back from a session on a task read nothing back")
	}

	filed, ok := cmd().(sessionFiledMsg)
	if !ok {
		t.Fatalf("reading the session back answered %T", cmd())
	}

	if asked.ID != "ACME-2662" || engine != "claude" || !since.Equal(started) {
		t.Errorf("the port was asked about %q on %q since %v, want the session that just ended", asked.ID, engine, since)
	}

	after, _ := advance(t, m, filed)
	if band := ansi.Strip(after.bandLine(100)); !strings.Contains(band, "4") || !strings.Contains(band, "ACME-2662") {
		t.Errorf("the band says %q, want how much of the session went into the task", band)
	}
}

// TestASessionThatSaidNothingIsNotAnnounced: a reader who opened a terminal
// to run two commands had no conversation, and a window reporting that is a
// window talking about its own plumbing.
func TestASessionThatSaidNothingIsNotAnnounced(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.say("the session on ACME-2662 ended")

	after, cmd := advance(t, m, sessionFiledMsg{ID: "ACME-2662"})
	if cmd != nil {
		t.Errorf("filing nothing asked for %v", cmd)
	}

	if band := ansi.Strip(after.bandLine(100)); !strings.Contains(band, "ended") {
		t.Errorf("the band says %q, want the sentence about the session left alone", band)
	}
}

// TestASessionThatCouldNotBeReadBackSaysSo, because the alternative is a
// conversation that quietly is not in the record and a reader who finds out
// a week later.
func TestASessionThatCouldNotBeReadBackSaysSo(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	after, _ := advance(t, m, sessionFiledMsg{ID: "ACME-2662", Err: errors.New("transcript unreadable")})
	if band := ansi.Strip(after.bandLine(100)); !strings.Contains(band, "transcript unreadable") {
		t.Errorf("the band says %q, want why the session did not come back", band)
	}
}

// TestAWindowWithNoWayToReadASessionBackStillComesBack from one: the port
// is optional, and a window handed none is the window that existed before
// anything read a session back.
func TestAWindowWithNoWayToReadASessionBackStillComesBack(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.FileSession = nil

	after, cmd := advance(t, m, cliEndedMsg{Engine: "claude", Repo: "/repo", Task: view.Task{ID: "ACME-2662"}})
	if cmd != nil {
		t.Errorf("a window with no port asked for %v", cmd)
	}

	if !strings.Contains(ansi.Strip(after.bandLine(100)), "ACME-2662") {
		t.Error("the window said nothing about the session that ended")
	}
}

// TestFilingRefreshesTheTaskTheReaderIsStandingOn.
//
// The notes tab draws the entries this window is already holding, so a
// conversation written into the record while the reader is looking at that
// very task would be in the record and not on the screen until they walked
// away and came back.
func TestFilingRefreshesTheTaskTheReaderIsStandingOn(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2701")
	m.screen, m.detail = screenDetail, "ACME-2701"

	if _, cmd := advance(t, m, sessionFiledMsg{ID: "ACME-2701", Turns: 2}); cmd == nil {
		t.Fatal("the record was not asked for again after a session went into the task on screen")
	}

	if _, cmd := advance(t, m, sessionFiledMsg{ID: "ACME-2662", Turns: 2}); cmd != nil {
		t.Error("the record of the task on screen was re-read for a session on another one")
	}
}
