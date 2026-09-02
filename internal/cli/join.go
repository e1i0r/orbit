package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// joinRepo opens a checkout of another repository for a task, which is that
// repository joining it.
//
// It is written for a caller with no arguments to spare: an engine halfway
// through a phase, in a worktree, that has just found out the change it is
// making needs the API as well as the app. The task comes from the
// environment the run put it in, so what the model has to get right is one
// word — the name of the repository — and the answer is the directory to
// work in.
//
// A reader typing it by hand names the task with -task, from a directory
// inside any repository of the workspace.
func joinRepo(ctx Context, args []string) error {
	fs := flag.NewFlagSet("join", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "a repository of the workspace")

	id := fs.String("task", os.Getenv(task.IDEnv), "the task the repository joins")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	name := fs.Arg(0)
	if name == "" {
		return fmt.Errorf("%s", ctx.printer().T("join.needs_a_name",
			"join needs the name of a repository; `orbit repos` lists them"))
	}

	if *id == "" {
		return fmt.Errorf("%s", ctx.printer().T("join.needs_a_task",
			"join needs a task: pass -task, or run it inside a phase, where {env} is already set",
			words.Arg{Name: "env", Value: task.IDEnv}))
	}

	wt, err := joined(*dir, *id, name)
	if err != nil {
		logger.Error("cli/join", "join %q to task %q failed: %v", name, *id, err)
		return err
	}

	logger.Info("cli/join", "%s joined task %s at %s", name, *id, wt)
	fmt.Fprintf(ctx.Out, "%s\n", wt)

	return nil
}

// joined resolves the name against the workspace and opens the checkout.
//
// The repository the command was run from is the one thing here that is not
// the task's own, and it is what the name is resolved against: its workspace
// is the set of checkouts on offer. A phase of a task that has joined
// nothing runs in a directory that is no repository at all, so that door is
// allowed to come back empty — the names then come from the repositories
// Orbit has a record of, which is the whole of where such a task can go.
func joined(dir, id, name string) (string, error) {
	s, r, err := openMaybe(dir, false)
	if err != nil {
		return "", err
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		return "", err
	}

	found, err := task.Joinable(s, r, name)
	if err != nil {
		return "", err
	}

	return task.Join(s, t, found)
}
