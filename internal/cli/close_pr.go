package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
)

// closingComment is what Orbit leaves on a pull request it closes. It lives
// here rather than in internal/repo because that package runs git and gh and
// does not write English.
const closingComment = "Closed from Orbit."

func closePR(ctx Context, args []string) error {
	fs := flag.NewFlagSet("close-pr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	if len(fs.Args()) < 1 {
		return fmt.Errorf("close-pr needs the task identifier")
	}

	taskID := fs.Args()[0]

	s, r, err := openBoth(*dir)
	if err != nil {
		logger.Error("cli/close-pr", "open repository %q failed: %v", *dir, err)
		return err
	}

	wtDir, err := s.WorktreeDir(r.Path, taskID)
	if err != nil {
		logger.Error("cli/close-pr", "get worktree for task %q failed: %v", taskID, err)
		return err
	}

	branch := "orbit/" + taskID

	if err := r.ClosePR(wtDir, branch, closingComment); err != nil {
		logger.Error("cli/close-pr", "gh pr close failed: %v", err)
		return fmt.Errorf("close pull request for %q failed: %w", taskID, err)
	}

	logger.Info("cli/close-pr", "closed pull request for task %s on branch %s", taskID, branch)
	fmt.Fprintf(ctx.Out, "Pull Request closed: %s\n", branch)

	return nil
}
