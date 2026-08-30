package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// reconcile closes the records of runs whose processes are gone.
func reconcile(ctx Context, args []string) error {
	fs := flag.NewFlagSet("reconcile", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the tasks are against")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	s, r, err := openBoth(*dir)
	if err != nil {
		logger.Error("cli/reconcile", "open repository %q failed: %v", *dir, err)
		return err
	}

	ids := []string{fs.Arg(0)}
	if fs.Arg(0) == "" {
		if ids, err = task.List(s, r); err != nil {
			logger.Error("cli/reconcile", "list tasks in repo %q failed: %v", r.Name, err)
			return err
		}
	}

	closed := 0

	var errs []error

	for _, id := range ids {
		wrote, err := task.Reconcile(s, task.Task{ID: id, Repo: r})
		if err != nil {
			logger.Error("cli/reconcile", "reconcile task %q in repo %q failed: %v", id, r.Name, err)
			errs = append(errs, err)

			continue
		}

		if wrote {
			closed++

			logger.Info("cli/reconcile", "reconciled and marked abandoned: task %s in repo %s", id, r.Name)
			fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("reconcile.abandoned",
				"{id} was abandoned; its record says so now",
				words.Arg{Name: "id", Value: id}))
		}
	}

	if closed == 0 && len(errs) == 0 {
		fmt.Fprintln(ctx.Out, ctx.printer().T("reconcile.all_accounted", "every run is accounted for"))
	}

	return errors.Join(errs...)
}
