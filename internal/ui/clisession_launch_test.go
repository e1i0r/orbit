package ui

// clisession_coverage_test.go covers the two gestures around handing the
// terminal to an interactive CLI: building the command line to exec, and
// reading what the process said when it came back.
//
// Nothing here starts a real process. tea.ExecProcess (see charm.land/
// bubbletea/v2's exec.go) only ever wraps the *exec.Cmd into a message that
// the program's own runtime would run — calling the tea.Cmd this package
// hands back merely builds that message, exactly the way advance() calls
// every other tea.Cmd in this suite.

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

func TestLaunchInteractiveCLIPicksTheEngineAndRepo(t *testing.T) {
	// 1. Default engine, cursor on a task row: the row's own repo is used.
	m, _ := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2662")
	next, cmd := m.launchInteractiveCLI()
	after := asModel(t, next)
	if cmd == nil {
		t.Fatal("launchInteractiveCLI answered with no command")
	}
	wantBand(t, after, "opening interactive session")
	wantBand(t, after, "claude")

	// 2. A knob naming another engine is said instead of the default.
	m2, _ := testModel(t, 100, 30)
	m2 = onRow(t, m2, "ACME-2662")
	m2.knobs.Engine = "codex"
	next2, cmd2 := m2.launchInteractiveCLI()
	after2 := asModel(t, next2)
	if cmd2 == nil {
		t.Fatal("launchInteractiveCLI answered with no command")
	}
	wantBand(t, after2, "codex")

	// 3. Cursor on a band header: the row is not a task, so the fallback to
	// the board's repo list decides the directory instead.
	m3, _ := testModel(t, 100, 30)
	m3 = at(t, m3, boardHeaderBand(t, m3), true)
	m3.board.RepoList = []board.RepoInfo{{Name: "svc", Path: "/tmp/svc"}}
	_, cmd3 := m3.launchInteractiveCLI()
	if cmd3 == nil {
		t.Fatal("launchInteractiveCLI on a header row answered with no command")
	}

	// 4. Nothing selected and no repo list at all: repoDir ends up empty,
	// and the window still hands back a command rather than doing nothing.
	m4, _ := testModel(t, 100, 30)
	m4.cursor = -1
	_, cmd4 := m4.launchInteractiveCLI()
	if cmd4 == nil {
		t.Fatal("launchInteractiveCLI with nothing selected answered with no command")
	}
}

// boardHeaderBand is the band of the first head row the fixture board draws,
// which is what test 3 above needs to land the fallback branch of
// launchInteractiveCLI rather than a task's own repo.
func boardHeaderBand(t *testing.T, m Model) view.Band {
	t.Helper()
	for _, r := range m.rows() {
		if r.head {
			return r.band
		}
	}
	t.Fatal("no band header row in the body")
	return view.ToDo
}

func TestHandleCLIEndedOpensTheConfirmOnASuccessfulRun(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	next, cmd := m.handleCLIEnded(cliEndedMsg{Engine: "claude", Repo: "payments"})
	if cmd != nil {
		t.Errorf("handleCLIEnded on a clean exit produced a command, want none")
	}
	if next.confirm != confirmPostCliTask || next.confirmID != "payments" {
		t.Errorf("confirm=%v confirmID=%q, want confirmPostCliTask on %q", next.confirm, next.confirmID, "payments")
	}
	wantBand(t, next, "create a task in Orbit")
}

func TestHandleCLIEndedTreatsAnExitErrorAsAnOrdinaryEnding(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	next, cmd := m.handleCLIEnded(cliEndedMsg{Engine: "claude", Repo: "app", Err: &exec.ExitError{}})
	if cmd != nil {
		t.Errorf("handleCLIEnded on an *exec.ExitError produced a command, want none")
	}
	if next.confirm != confirmPostCliTask || next.confirmID != "app" {
		t.Errorf("confirm=%v confirmID=%q, want confirmPostCliTask on %q", next.confirm, next.confirmID, "app")
	}
}

func TestHandleCLIEndedSaysWhenTheProcessNeverRan(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	next, cmd := m.handleCLIEnded(cliEndedMsg{Engine: "claude", Repo: "payments", Err: errors.New("exec: \"claude\": executable file not found in $PATH")})
	if cmd != nil {
		t.Errorf("handleCLIEnded on a launch failure produced a command, want none")
	}
	if next.confirm != confirmNone {
		t.Errorf("confirm=%v, want confirmNone: a process that never ran opened nothing", next.confirm)
	}
	wantBand(t, next, "error running claude")
}
