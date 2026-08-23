package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/task"
)

// runTask walks a task through a flow. It is the one command that spends
// money: it invokes the real engine.
//
// Ctrl-C and SIGTERM stop it, and stopping it writes phase.cancelled and
// task.cancelled with whatever the engine had printed first — a run that is
// interrupted says so in its own record rather than leaving phase.started as
// the last line, which reads for ever as a task still running. -timeout is
// the same stop on a clock, and it is recorded as its own fact: a run you
// stopped is done with, a run that outlived its deadline wants you.
//
// The default is no timeout, because there is no honest default. Phases run
// for minutes or for an hour depending on the task, and a number picked here
// would end real work in the middle for the sake of tidiness. A run that is
// wedged is a run somebody can see in the window, and `orbit cancel` is one
// line away.
func runTask(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	name := fs.String("flow", "task", "which flow to walk")
	timeout := fs.Duration("timeout", 0, "stop the run after this long, e.g. 45m; zero waits for as long as it takes")
	if err := parse(fs, args, out); err != nil {
		return err
	}
	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("run needs the id of a task")
	}

	s, r, err := openBoth(*dir)
	if err != nil {
		return err
	}
	f, err := flow.Builtin(*name)
	if err != nil {
		return err
	}
	t, err := task.Load(s, r, id)
	if err != nil {
		return err
	}
	// Installed here, after everything that can be wrong about the command
	// itself has been found: a mistyped id should not go through a signal
	// handler on its way to being reported.
	//
	// SIGTERM as well as interrupt, because `orbit cancel` sends SIGTERM,
	// and the two gestures — Ctrl-C at the terminal and cancel from the
	// window — have to arrive at the same place and be written down the
	// same way. stop is deferred so the handler is taken off again: a
	// process that keeps swallowing signals after it is done cannot be
	// killed by the ordinary means.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *timeout > 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, *timeout)
		defer done()
	}

	engines := map[string]engine.Engine{"claude": engine.NewClaude()}
	if err := task.Run(ctx, s, t, f, engines); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s finished\n", id)
	return nil
}
