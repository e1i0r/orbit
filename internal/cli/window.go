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

// canResume is whether one named engine can carry on a session it started
// before.
//
// It answers about one engine because the window asks about one task, and
// because the refusal it produces names an engine. It used to be an AND over
// every engine configured — a standing fact for the whole program — and that
// was wrong in a way that only shows up with two engines: if either of them
// could not resume, t was refused on every task, and each task was told that
// its own engine was the one that could not. The name comes off the task,
// which is where the record keeps it.
//
// A name nothing is configured for is false rather than an error. The window
// is drawing a key's reason, not validating a configuration, and a task
// recorded against an engine this build no longer has is a task whose session
// cannot be resumed by anything here — which is exactly what false says. So
// is a nil in the map, and so is the empty name a task that has never run
// carries.
func canResume(engines map[string]engine.Engine, name string) bool {
	e, ok := engines[name]
	return ok && e != nil && e.CanResume()
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
