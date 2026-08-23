package cli

// window.go answers the two questions the window asks of this layer about
// engines, and it is here rather than in internal/ui because internal/ui
// cannot import internal/engine and never will: the window draws what an
// engine can do, and knowing which engines exist is the composition root's
// job. Both answers travel as ports on ui.Options.
//
// Nothing here runs anything. takeCommand builds a command line and hands it
// back; the window is what suspends the terminal for it.

import (
	"fmt"
	"os/exec"

	"github.com/e1i0r/orbit/internal/engine"
)

// canResume is whether the engines this program is configured with can carry
// on a session they started before.
//
// Every one of them has to, and not merely one: the answer is a standing
// fact the window shows beside a task whose engine it has not looked up, and
// an optimistic answer would offer t on a task the engine behind it cannot
// resume — a key that is offered and then refused is worse than one that was
// greyed out with its reason.
//
// No engines at all is false rather than vacuously true, for the same
// reason: nothing configured is nothing that can resume anything.
func canResume(engines map[string]engine.Engine) bool {
	if len(engines) == 0 {
		return false
	}
	for _, e := range engines {
		if e == nil || !e.CanResume() {
			return false
		}
	}
	return true
}

// takeCommand builds the interactive session the window suspends itself for
// when the reader presses t.
//
// --fork-session is not optional and not a nicety. A resumed session that is
// not forked writes into the session the runner recorded, so a conversation
// a person has by hand would land in the middle of the transcript a phase is
// going to be judged by — and the run's own record would then describe a
// session that no longer says what it said when the phase ran. Forking gives
// the interactive session a new id, and the runner's transcript is never
// touched.
//
// dir is the task's worktree, and it is set rather than inherited because a
// session opened in whatever folder orbit was started from would be reading
// and writing the wrong checkout. The refusal for a worktree somebody else
// is writing in is not here — it is in the window, where it can name the key
// to press first.
//
// A task with no session id is not an error: an engine that does not report
// one is a fact about that engine, and the window says so in the reader's
// own language. A nil command with a nil error is that answer.
func takeCommand(eng engine.Engine, session, dir string) (*exec.Cmd, error) {
	if eng == nil {
		return nil, fmt.Errorf("taking the keyboard needs an engine")
	}
	if !eng.CanResume() {
		return nil, fmt.Errorf("%s cannot resume a session", eng.Name())
	}
	if dir == "" {
		return nil, fmt.Errorf("taking the keyboard for a %s session needs a worktree to open it in", eng.Name())
	}
	if session == "" {
		return nil, nil
	}
	cmd := exec.Command(eng.Name(), "--resume", session, "--fork-session")
	cmd.Dir = dir
	return cmd, nil
}
