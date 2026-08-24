package ui

// recorder_test.go is the four closures the window is handed instead of a
// state root, written to record rather than to act.
//
// It is a file of its own because it is the only place in this package where
// a test decides what a port would have done, and because fixture_test.go is
// the board rather than the wiring. Nothing here writes a file, starts a
// process or reaches a network: the one port that produces a command line
// builds an *exec.Cmd by hand and hands it back unstarted, which is exactly
// what the window then asserts on and never runs.

import (
	"os/exec"

	"github.com/e1i0r/orbit/internal/view"
)

// fixtureWorktree is where the fixture's throwaway checkouts would be. It is
// under the fixture root and it is never created: the only thing any test
// asks of it is that a session would have been opened somewhere rather than
// in whatever directory orbit was started in.
const fixtureWorktree = "~/work/.orbit/worktrees/"

// fixtureSession is the prefix of the session id the fake engine would have
// recorded, so that a test reading a command line can tell one task's session
// from another's.
const fixtureSession = "session-"

// recorder records what a gesture asked for, so a Cmd can be executed in a
// test without anything being controlled, started, read or resumed. It is not
// named control because msg.go's control is the function it stands in for.
type recorder struct {
	id, word string // the control port: which task, which of the five words
	flow     string // the start port: which flow the dialog chose
	unread   int    // and the count it passed to the cap
	read     string // the mark-read port
	taken    string // the take-the-keyboard port
	err      error
}

// ports is the recorder as the window's Options, and a nil recorder as ports
// that answer without recording — which is what a rendering test wants and
// what a window opened with no gestures in it would have.
func (got *recorder) ports() Options {
	return Options{
		Control: func(t view.Task, word string) error {
			if got == nil {
				return nil
			}
			got.id, got.word = t.ID, word
			return got.err
		},
		Start: func(t view.Task, flowName string, unread int) (int, error) {
			if got == nil {
				return 0, nil
			}
			got.id, got.flow, got.unread = t.ID, flowName, unread
			// A pid rather than a zero, because the band says which process
			// a run was given and a zero there would read as "none".
			return 4242, got.err
		},
		MarkRead: func(t view.Task) error {
			if got == nil {
				return nil
			}
			got.read = t.ID
			return got.err
		},
		Take: func(t view.Task) (*exec.Cmd, error) {
			if got != nil {
				got.taken = t.ID
			}
			return sessionCommand(t), nil
		},
		// Flows is nil, which is the built-ins and nothing else — the same
		// answer a window opened without a state root gets, and the right
		// default for every test that is about something else. The two that
		// are about a reader's own flows hand the window a directory of
		// their own; userFlows in start_render_test.go is where that is.
	}
}

// sessionCommand is the command line internal/cli builds for t, as this
// package's tests need to be able to read it.
//
// It is assembled field by field rather than through exec.Command, which
// would look the program up on the test machine's PATH. Nothing here is ever
// run, so a path is a string; and a test that quietly depended on claude
// being installed would fail on one machine and pass on another.
func sessionCommand(t view.Task) *exec.Cmd {
	return &exec.Cmd{
		Path: "claude",
		Args: []string{"claude", "--resume", fixtureSession + t.ID, "--fork-session"},
		Dir:  fixtureWorktree + t.ID,
	}
}
