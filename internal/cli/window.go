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
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/ui"
	"github.com/e1i0r/orbit/internal/words"
)

// canResume is whether one named engine can carry on a session it started
// before.
// It answers about one engine because the window asks about one task, and
// because the refusal it produces names an engine. An AND over every engine
// configured — a standing fact for the whole program — is wrong in a way that
// only shows up with two engines: if either of them cannot resume, t is
// refused on every task, and each task is told that its own engine is the one
// that cannot. The name comes off the task,
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

// commandTable is what the window's palette shows of the command list: the
// name, the usage fragment and the description, plus the refusal with its
// reason for a command that makes no sense from inside.
//
// It is a copy in the window's shape rather than the table itself because
// ui.Command deliberately carries no Run — the window names a command, it
// does not call one — and because WindowOpens is not a refusal: those are
// commands the window will answer with screens of their own, and showing
// them greyed would say something untrue about them.
func commandTable() []ui.Command {
	cs := commands()

	out := make([]ui.Command, 0, len(cs))
	for _, c := range cs {
		uc := ui.Command{Name: c.Name, Args: c.Args, About: c.About}
		if c.InWindow == WindowRefuses {
			uc.Refused = true
			uc.Because = c.Because
		}

		// Only for the commands the window actually runs. A command it
		// answers with a screen has its own reason for not running here, and
		// telling a reader to type an id for it would send them back with
		// the id to the same refusal.
		uc.NeedsArgs = c.NeedsArgs && c.InWindow == WindowRuns

		out = append(out, uc)
	}

	return out
}

// doPort is how a named command is run from inside the window, and it is
// the same table `orbit` dispatches from — which is the whole constraint:
// no command exists in one entry point and not the other.
//
// The printer is resolved per run rather than captured once, because the
// reader can change language while the window is open; a refusal sentence
// frozen at startup would come back in a language nobody on screen speaks
// any more. The command's output goes to out whole, so the window can watch
// the work as it happens; its own rules about what to print where are the
// commands', unchanged.
func doPort(lang interface{ Language() string }) func(string, []string, io.Writer) error {
	return func(name string, args []string, out io.Writer) error {
		c, ok := lookup(name)
		if !ok {
			p := words.For(lang.Language())

			return errors.New(p.T("msg.no_such_command", "no such command: {name}",
				about(name)))
		}

		p := words.For(lang.Language())

		switch c.InWindow {
		case WindowRefuses:
			// The table owns the reason; this only delivers it.
			return errors.New(c.Because(p))
		case WindowOpens:
			// The screens these commands open are tasks 9 and 13, and
			// until they exist an honest nothing is better than a table
			// printed into a pane nobody asked for.
			return errors.New(p.T("msg.no_screen_yet",
				"{name} opens a screen this window does not have yet", about(name)))
		}

		return c.Run(Context{Out: out, Err: out, Words: p}, args)
	}
}

// about is the one substitution doPort's two sentences share.
func about(name string) words.Arg {
	return words.Arg{Name: "name", Value: name}
}
