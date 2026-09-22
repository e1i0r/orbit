package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

func mergePR(ctx Context, args []string) error {
	fs := flag.NewFlagSet("pr merge", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	if len(fs.Args()) < 1 {
		return needsTaskID(ctx, "pr merge")
	}

	p := ctx.printer()
	taskID := fs.Args()[0]

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		return fmt.Errorf("open repository %q: %w", *dir, err)
	}

	// Where the task was worked, and not where the reader happens to be
	// standing: Orbit is opened on the directory above the repositories, and
	// a task that reached into three of them has three pull requests to
	// merge.
	where, err := worked(s, r, taskID)
	if err != nil {
		return fmt.Errorf("read the repositories of task %q: %w", taskID, err)
	}

	branch := branchFor(s, r, taskID)

	for _, one := range where {
		wtDir, wtErr := s.WorktreeDir(one.Path, taskID)
		if wtErr != nil {
			logger.Error("cli/merge", "get worktree for task %q failed: %v", taskID, wtErr)
			return wtErr
		}

		// What GitHub says about the branch, before asking it to merge.
		//
		// gh refuses a pull request whose required checks have not passed
		// and prints a line of its own, which Orbit passed through as
		// "merging the pull request of FRA-128 failed: exit status 1". A
		// reader who pressed M had to go and look. Read first and the
		// refusal names which check is red, and how many are still going.
		//
		// A reading that could not be taken is not a refusal: gh may not
		// be installed on this machine or the repository may have no
		// remote, and neither is a reason to stop a merge that would have
		// worked. The merge itself still refuses in that case, the way it
		// always did.
		if red, waiting, ok := merging(one, wtDir, branch); ok {
			return refuseMerge(p, taskID, red, waiting)
		}

		if err := one.MergePR(wtDir, branch); err != nil {
			logger.Error("cli/merge", "gh pr merge failed: %v", err)

			return fmt.Errorf("%s: %w", p.T("merge.refused", "merging the pull request of {id} failed",
				words.Arg{Name: "id", Value: taskID}), err)
		}

		// Marked where it merged, like the row was opened where it opened:
		// a mark that will not write is warned about for the same reason.
		if werr := s.MarkPR(taskID, one.Path, store.PRMerged); werr != nil {
			logger.Warn("cli/merge", "mark the pull request of %q in %q merged: %v", taskID, one.Name, werr)
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

// merging reads the checks and says whether any of them stand in the way.
func merging(r repo.Repo, wtDir, branch string) (red, waiting []string, blocked bool) {
	runs, err := r.PRChecks(wtDir, branch)
	if err != nil {
		logger.Warn("cli/merge", "read the checks on %q: %v", branch, err)

		return nil, nil, false
	}

	red, waiting = repo.Blocking(runs)

	return red, waiting, len(red) > 0 || len(waiting) > 0
}

// refuseMerge is what a reader is told instead of gh's exit status.
func refuseMerge(p *words.Printer, taskID string, red, waiting []string) error {
	if len(red) > 0 {
		return errors.New(p.T("merge.checks_red",
			"{id} was not merged: {checks} failed on GitHub — fix them with the checks key, "+
				"or merge it there yourself if you mean to override them",
			words.Arg{Name: "id", Value: taskID},
			words.Arg{Name: "checks", Value: strings.Join(red, ", ")}))
	}

	return errors.New(p.T("merge.checks_pending",
		"{id} was not merged: {checks} still running on GitHub — try again when they land",
		words.Arg{Name: "id", Value: taskID},
		words.Arg{Name: "checks", Value: strings.Join(waiting, ", ")}))
}
