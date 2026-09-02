package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
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
		return needsTaskID(ctx, "cancel")
	}

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		logger.Error("cli/cancel", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		logger.Error("cli/cancel", "load task %q in %q failed: %v", id, r.Name, err)
		return err
	}

	if *now {
		if err := task.Kill(s, t); err != nil {
			logger.Error("cli/cancel", "kill task %q in %q failed: %v", id, r.Name, err)
			return err
		}

		logger.Warn("cli/cancel", "task %s in %s killed outright", id, r.Name)
		fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("cancel.killed",
			"{id} killed — run `orbit reconcile -repo {repo}` to close its record",
			words.Arg{Name: "id", Value: id}, words.Arg{Name: "repo", Value: *dir}))

		return nil
	}

	if err := task.Cancel(s, t); err != nil {
		logger.Error("cli/cancel", "cancel task %q in %q failed: %v", id, r.Name, err)
		return err
	}

	logger.Info("cli/cancel", "task %s in %s requested to stop gracefully", id, r.Name)
	fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("cancel.asked_to_stop", "{id} asked to stop",
		words.Arg{Name: "id", Value: id}))

	return nil
}
