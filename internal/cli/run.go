package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/task"
)

// runTask walks a task through a flow. It is the one command that spends
// money: it invokes the real engine.
//
// There is no timeout and no signal handling here yet. A hung engine hangs
// orbit, and Ctrl-C kills it with phase.started as the last thing in the
// record, which reads as a task still running. Cancellation belongs with the
// window that will own it.
func runTask(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	name := fs.String("flow", "task", "which flow to walk")
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
	engines := map[string]engine.Engine{"claude": engine.NewClaude()}
	if err := task.Run(context.Background(), s, t, f, engines); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s finished\n", id)
	return nil
}
