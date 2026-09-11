package cli

// The names a command answered to before its verb joined a family.
//
// A rename that breaks every script somebody wrote is not a rename; it is a
// removal with something new standing next to it. So the old spelling still
// runs, it says what to type instead, and the help screen describes it as
// what it is rather than as a second verb.
//
// Which old name became which is internal/verb's to say. Everything here
// reads that rather than carrying a list of its own — the four ways in
// disagreeing about a name is the whole reason that package exists.

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// underItsOldName is the same command, spelled the way it used to be.
func underItsOldName(v verb.Verb) Command {
	c := commandFor(v)
	c.Name, c.About = v.Was, oldName(v.Was)
	c.Run = wasCalled(v.Was, c.Run)

	return c
}

// oldName is what it says about itself on the help screen: not what the verb
// does — the line above it is the new spelling and says that — but where the
// name went. Two lines describing the same work identically would be a
// screen that looks like Orbit has two of everything.
func oldName(was string) func(*words.Printer) string {
	return func(p *words.Printer) string {
		v, declared := verb.One(was)
		if !declared {
			return ""
		}

		return p.T("cmd.was_called", "the old name for orbit {new}",
			words.Arg{Name: "new", Value: v.Path()})
	}
}

// wasCalled says once, before the command runs, what to type instead.
//
// On the error stream and not with the answer: a notice mixed into what a
// reader piped somewhere is a notice that breaks the pipe. The browser and a
// tool call have one channel and get the same sentence out of internal/verb,
// which is where all three read what the old name was.
//
// The verb is looked up by the old name rather than named here, so a command
// whose verb has stopped claiming that name says nothing rather than saying
// something that is no longer true.
func wasCalled(
	was string, run func(Context, []string) error,
) func(Context, []string) error {
	return func(ctx Context, args []string) error {
		if v, declared := verb.One(was); declared && v.Was == was {
			fmt.Fprintln(ctx.Err, ctx.printer().T("verb.was_called",
				"{old} is now {new}, and the old name still works",
				words.Arg{Name: "old", Value: was},
				words.Arg{Name: "new", Value: v.Path()}))
		}

		return run(ctx, args)
	}
}
