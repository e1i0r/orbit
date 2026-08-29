package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
)

func mergePR(ctx Context, args []string) error {
	fs := flag.NewFlagSet("merge", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	if len(fs.Args()) < 1 {
		return fmt.Errorf("merge needs the task identifier")
	}

	taskID := fs.Args()[0]

	s, r, err := openBoth(*dir)
	if err != nil {
		logger.Error("cli/merge", "open repository %q failed: %v", *dir, err)
		return err
	}

	wtDir, err := s.WorktreeDir(r.Path, taskID)
	if err != nil {
		logger.Error("cli/merge", "get worktree for task %q failed: %v", taskID, err)
		return err
	}

	branch := "orbit/" + taskID

	if err := r.MergePR(wtDir, branch); err != nil {
		logger.Error("cli/merge", "gh pr merge failed: %v", err)
		return fmt.Errorf("merge pull request for %q failed: %w", taskID, err)
	}

	logger.Info("cli/merge", "merged pull request for task %s on branch %s", taskID, branch)
	fmt.Fprintf(ctx.Out, "Pull Request merged: %s\n", branch)

	return nil
}
