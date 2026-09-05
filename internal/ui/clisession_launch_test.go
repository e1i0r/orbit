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

	wantBand(t, next, "make a task from this session")
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

// The window hands the port the task the cursor is on and the directory it
// believes the session belongs in — and that directory is the repository's
// path, not its name. The name is a column and not a place: a session opened
// on it starts in a directory that does not exist, or, worse, in one that
// happens to.
func TestLaunchInteractiveCLIAsksThePortForTheSession(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	for i := range m.board.Tasks {
		m.board.Tasks[i].RepoPath = "/checkouts/" + m.board.Tasks[i].Repo
	}

	m = onRow(t, m, "ACME-2662")

	var (
		gotTask           view.Task
		gotEngine, gotDir string
	)

	m.opts.Open = func(task view.Task, engineName, dir string) (*exec.Cmd, error) {
		gotTask, gotEngine, gotDir = task, engineName, dir
		return exec.Command("true"), nil
	}

	next, cmd := m.launchInteractiveCLI()
	if cmd == nil {
		t.Fatal("launchInteractiveCLI answered with no command")
	}

	if gotTask.ID != "ACME-2662" {
		t.Errorf("the port was asked about %q, want the task the cursor is on", gotTask.ID)
	}

	if gotEngine != "claude" {
		t.Errorf("the port was asked for %q, want the engine the knob names", gotEngine)
	}

	if gotDir != "/checkouts/payments" {
		t.Errorf("the port was given %q, want the repository's path", gotDir)
	}

	wantBand(t, asModel(t, next), "opening interactive session")
}

// A session that could not be built is said and not started: the reader
// pressed a key and the terminal stayed where it was, so the window owes
// them the reason.
func TestLaunchInteractiveCLISaysWhenTheSessionCouldNotBeBuilt(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2662")
	m.opts.Open = func(view.Task, string, string) (*exec.Cmd, error) {
		return nil, errors.New("opening an interactive session needs an engine")
	}

	next, cmd := m.launchInteractiveCLI()
	if cmd != nil {
		t.Error("a session that could not be built was started anyway")
	}

	wantBand(t, asModel(t, next), "error running claude")
}

// A window with no port opens the engine itself, which is what it did before
// there was one.
func TestOpenSessionWithoutAPortIsABareEngine(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	cmd, err := m.openSession(view.Task{}, "codex", "/checkouts/app")
	if err != nil {
		t.Fatalf("openSession: %v", err)
	}

	if len(cmd.Args) != 1 {
		t.Errorf("args = %v, want the engine and nothing else", cmd.Args)
	}

	if cmd.Dir != "/checkouts/app" {
		t.Errorf("Dir = %q, want the directory it was given", cmd.Dir)
	}
}
