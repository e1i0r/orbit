package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/task"
)

// reconcile closes the records of runs whose processes are gone.
//
// A run that was killed outright, or a machine that went down mid-phase,
// leaves a log whose last event is phase.started — which reads for ever as a
// task still running, because the process that could have said otherwise was
// not given the chance. This is the reader that says otherwise, and it is
// the same function the window calls when it opens. Naming a task does one;
// naming none does every task in the repository, which is the usual way to
// use it after a laptop has been closed on a run.
//
// A task that cannot be reconciled does not stop the others: the point of
// the sweep is the tasks it can close, and one damaged marker holding up the
// rest would make the command useless exactly when it is needed.
func reconcile(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("reconcile", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the tasks are against")
	if err := parse(fs, args, out); err != nil {
		return err
	}
	s, r, err := openBoth(*dir)
	if err != nil {
		return err
	}

	ids := []string{fs.Arg(0)}
	if fs.Arg(0) == "" {
		if ids, err = task.List(s, r); err != nil {
			return err
		}
	}

	closed := 0
	var errs []error
	for _, id := range ids {
		wrote, err := task.Reconcile(s, task.Task{ID: id, Repo: r})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if wrote {
			closed++
			fmt.Fprintf(out, "%s was abandoned; its record says so now\n", id)
		}
	}
	if closed == 0 && len(errs) == 0 {
		fmt.Fprintln(out, "every run is accounted for")
	}
	return errors.Join(errs...)
}
