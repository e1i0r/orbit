package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
)

func createPR(ctx Context, args []string) error {
	fs := flag.NewFlagSet("pr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}
	if len(fs.Args()) < 1 {
		return fmt.Errorf("pr needs the task identifier")
	}
	taskID := fs.Args()[0]

	s, r, err := openBoth(*dir)
	if err != nil {
		logger.Error("cli/pr", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, taskID)
	if err != nil {
		logger.Error("cli/pr", "load task %q failed: %v", taskID, err)
		return err
	}

	wtDir, err := s.WorktreeDir(r.Path, taskID)
	if err != nil {
		logger.Error("cli/pr", "get worktree for task %q failed: %v", taskID, err)
		return err
	}
	branch := "orbit/" + taskID
	commitMsg := fmt.Sprintf("feat(%s): %s", taskID, t.Text)
	if len(commitMsg) > 72 {
		commitMsg = commitMsg[:72]
	}

	if err := r.CommitWorktree(wtDir, commitMsg); err != nil {
		logger.Error("cli/pr", "commit worktree %q failed: %v", wtDir, err)
		return err
	}

	if err := r.PushBranch(wtDir, branch); err != nil {
		logger.Error("cli/pr", "push branch %q failed: %v", branch, err)
		return err
	}

	body := fmt.Sprintf("## Orbit Task: %s\n\n%s\n\nGenerated automatically by Orbit.", taskID, t.Text)
	title := fmt.Sprintf("%s: %s", taskID, t.Text)
	if len(title) > 72 {
		title = title[:72]
	}
	prURL, err := r.CreatePR(wtDir, title, body, branch)
	if err != nil {
		logger.Error("cli/pr", "gh pr create failed: %v", err)
		return fmt.Errorf("branch %q pushed, but gh pr create failed: %w", branch, err)
	}

	logger.Info("cli/pr", "created pull request %s for task %s", prURL, taskID)
	fmt.Fprintf(ctx.Out, "Pull Request created: %s\n", prURL)
	return nil
}
