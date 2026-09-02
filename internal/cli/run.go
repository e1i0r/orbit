package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
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
//
// The gate is a real one even here, where there is no window: a run started
// from a terminal is still a run the reader can pause from another terminal,
// and a phase whose flow asks to wait still waits. One second between looks
// at the control file is a tenth of a human reaction and is only paid at a
// phase boundary that is already stopped — a run nobody is holding never
// reaches the loop that polls.
//
// -flow is an override and not the choice. The choice was made when the
// task was written down, and this flag exists because re-running a task
// through something else is a real thing to do the second time it fails.
// It stops being invisible: the run records the flow it walked, so a
// difference between the two is a decision somebody can see rather than a
// bug that takes an afternoon.
func runTask(ctx Context, args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	name := fs.String("flow", "", "walk this flow instead of the one the task was written against")

	timeout := fs.Duration("timeout", 0, "stop the run after this long, e.g. 45m; zero waits for as long as it takes")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	id := fs.Arg(0)
	if id == "" {
		return needsTaskID(ctx, "run")
	}

	// As in `orbit new`: a -repo the reader typed has to open, and the
	// default may come back empty. A run started by the window passes the
	// task's repository when it has one and nothing when it has not, so the
	// absence here is what carries "this task is against no repository"
	// across the process boundary.
	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		logger.Error("cli/run", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		logger.Error("cli/run", "load task %q in %q failed: %v", id, r.Name, err)
		return err
	}
	// The task's own flow, unless this command overrode it — and the flow
	// this program ships for a task written before the flow was recorded at
	// all. Not the settings default: that is what the *next* task written
	// gets, and applying it here would change what an old task walks
	// because a setting moved after it was written.
	chosen := *name
	if chosen == "" {
		chosen = t.Flow
	}

	if chosen == "" {
		chosen = flow.Default
	}

	f, err := flow.Resolve(s, chosen)
	if err != nil {
		logger.Error("cli/run", "resolve flow %q for task %q failed: %v", chosen, id, err)
		return err
	}
	// Installed here, after everything that can be wrong about the command
	// itself has been found: a mistyped id should not go through a signal
	// handler on its way to being reported.
	//
	// SIGTERM as well as interrupt, because `orbit cancel` sends SIGTERM,
	// and the two gestures — Ctrl-C at the terminal and cancel from the
	// window — have to arrive at the same place and be written down the
	// same way.
	signalled, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// The first signal is Orbit's to handle. The second is not.
	restoreOnCancel(signalled, stop)

	// running is the context the run itself lives in, and it is not ctx:
	// that name belongs to the command's own Context here, and one letter
	// between a deadline and a pair of writers is not a distinction worth
	// resting on.
	running := signalled

	if *timeout > 0 {
		var done context.CancelFunc

		running, done = context.WithTimeout(running, *timeout)
		defer done()
	}

	engines := newEngines()

	logger.Info("cli/run", "starting task %s in repo %s on flow %s (timeout=%v)", id, r.Name, chosen, *timeout)

	if err := task.Run(running, s, t, f, engines, task.FileGate(s, time.Second)); err != nil {
		logger.Error("cli/run", "task %s execution failed: %v", id, err)
		return err
	}

	logger.Info("cli/run", "task %s execution finished successfully", id)
	fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("run.finished", "{id} finished",
		words.Arg{Name: "id", Value: id}))

	return nil
}

// restoreOnCancel hands the signals back to the operating system as soon as
// the run is stopping, rather than when it has stopped.
//
// signal.NotifyContext cancels its context on the first signal and then goes
// on relaying: the handler stays installed until stop is called, and the
// deferred stop in runTask only runs after task.Run has returned. Everything
// between the two is the unwind — writing phase.cancelled, waiting on an
// engine that is taking its time about dying — and it is exactly when
// somebody presses Ctrl-C a second time. Swallowed there, the second one
// leaves `kill -9` as the only way out of a run that will not end.
//
// So: one goroutine, whose whole job is to take the handler off the moment
// the context is done, which restores the default disposition — the next
// interrupt kills the process outright, which is what pressing it twice
// means. stop is idempotent, so the deferred call is still correct, and on a
// run that is never signalled that deferred call is what ends this goroutine.
func restoreOnCancel(ctx context.Context, stop func()) {
	go func() {
		<-ctx.Done()
		stop()
	}()
}
