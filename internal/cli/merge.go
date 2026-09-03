package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

func mergePR(ctx Context, args []string) error {
	fs := flag.NewFlagSet("merge", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	if len(fs.Args()) < 1 {
		return needsTaskID(ctx, "merge")
	}

	p := ctx.printer()
	taskID := fs.Args()[0]

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		logger.Error("cli/merge", "open repository %q failed: %v", *dir, err)
		return err
	}

	// Where the task was worked, and not where the reader happens to be
	// standing: Orbit is opened on the directory above the repositories, and
	// a task that reached into three of them has three pull requests to
	// merge.
	where, err := worked(s, r, taskID)
	if err != nil {
		logger.Error("cli/merge", "read the repositories of task %q failed: %v", taskID, err)
		return err
	}

	branch := "orbit/" + taskID

	for _, one := range where {
		wtDir, wtErr := s.WorktreeDir(one.Path, taskID)
		if wtErr != nil {
			logger.Error("cli/merge", "get worktree for task %q failed: %v", taskID, wtErr)
			return wtErr
		}

		if err := one.MergePR(wtDir, branch); err != nil {
			logger.Error("cli/merge", "gh pr merge failed: %v", err)

			return fmt.Errorf("%s: %w", p.T("merge.refused", "merging the pull request of {id} failed",
				words.Arg{Name: "id", Value: taskID}), err)
		}
	}

	// Written down where the merge happened rather than inferred later from
	// a branch that is gone: a branch disappears for three other reasons
	// and only one of them is delivery, and what a digest counts as landed
	// has to be something somebody did.
	if t, loadErr := task.Load(s, r, taskID); loadErr == nil {
		if err := task.Merged(s, t, where[0].Name, branch); err != nil {
			logger.Warn("cli/merge", "write down that %q was merged: %v", taskID, err)
		}
	}

	logger.Info("cli/merge", "merged pull request for task %s on branch %s", taskID, branch)
	fmt.Fprintf(ctx.Out, "%s\n", p.T("merge.done", "pull request merged: {branch}",
		words.Arg{Name: "branch", Value: branch}))

	return nil
}
