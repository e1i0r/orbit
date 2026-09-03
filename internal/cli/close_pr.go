package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/words"
)

// closingComment is what Orbit leaves on a pull request it closes. It lives
// here rather than in internal/repo because that package runs git and gh and
// does not write English.
//
// It is asked of the catalogue like every other sentence orbit writes. The
// comment is read on GitHub rather than in the terminal, and that is a
// reason to translate it and not a reason to leave it: whoever closed the
// task from a Spanish cockpit is the one who will read it back.
func closingComment(ctx Context) string {
	return ctx.printer().T("close_pr.comment", "Closed from Orbit.")
}

func closePR(ctx Context, args []string) error {
	fs := flag.NewFlagSet("close-pr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	if len(fs.Args()) < 1 {
		return needsTaskID(ctx, "close-pr")
	}

	p := ctx.printer()
	taskID := fs.Args()[0]

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		logger.Error("cli/close-pr", "open repository %q failed: %v", *dir, err)
		return err
	}

	where, err := worked(s, r, taskID)
	if err != nil {
		logger.Error("cli/close-pr", "read the repositories of task %q failed: %v", taskID, err)
		return err
	}

	branch := "orbit/" + taskID

	for _, one := range where {
		wtDir, wtErr := s.WorktreeDir(one.Path, taskID)
		if wtErr != nil {
			logger.Error("cli/close-pr", "get worktree for task %q failed: %v", taskID, wtErr)
			return wtErr
		}

		if err := one.ClosePR(wtDir, branch, closingComment(ctx)); err != nil {
			logger.Error("cli/close-pr", "gh pr close failed: %v", err)

			return fmt.Errorf("%s: %w", p.T("close_pr.refused", "closing the pull request of {id} failed",
				words.Arg{Name: "id", Value: taskID}), err)
		}
	}

	logger.Info("cli/close-pr", "closed pull request for task %s on branch %s", taskID, branch)
	fmt.Fprintf(ctx.Out, "%s\n", p.T("close_pr.closed", "pull request closed: {branch}",
		words.Arg{Name: "branch", Value: branch}))

	return nil
}
