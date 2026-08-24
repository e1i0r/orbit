package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/task"
)

// cancelTask stops a run that is under way.
//
// Two gestures, because there are two situations. Without -now it asks: the
// run is signalled, it stops the engine, and it writes down that it was
// cancelled and what the phase had printed first. With -now it insists: the
// run and everything it started are killed outright, which is the answer for
// an engine that has stopped listening, and it leaves the record saying
// whatever it said — nothing written by a process being killed can be
// relied on. `orbit reconcile` is what closes that record afterwards, and
// the message says so rather than leaving the reader to find out.
func cancelTask(ctx Context, args []string) error {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	now := fs.Bool("now", false, "kill the run and everything it started, without waiting for it to write anything down")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}
	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("cancel needs the id of a task")
	}

	s, r, err := openBoth(*dir)
	if err != nil {
		return err
	}
	t, err := task.Load(s, r, id)
	if err != nil {
		return err
	}
	if *now {
		if err := task.Kill(s, t); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Out, "%s killed — run `orbit reconcile -repo %s` to close its record\n", id, *dir)
		return nil
	}
	if err := task.Cancel(s, t); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Out, "%s asked to stop\n", id)
	return nil
}
