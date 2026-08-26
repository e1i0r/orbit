package cli

import (
	"flag"
	"fmt"
	"io"

	"strings"

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

	firstLine := strings.SplitN(strings.TrimSpace(t.Text), "\n", 2)[0]
	commitMsg := fmt.Sprintf("feat(%s): %s", taskID, firstLine)
	if len(commitMsg) > 72 {
		if idx := strings.LastIndex(commitMsg[:72], " "); idx > 20 {
			commitMsg = commitMsg[:idx]
		} else {
			commitMsg = commitMsg[:72]
		}
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
	title := fmt.Sprintf("%s: %s", taskID, firstLine)
	if len(title) > 90 {
		if idx := strings.LastIndex(title[:90], " "); idx > 20 {
			title = title[:idx] + "..."
		} else {
			title = title[:87] + "..."
		}
	}
	prURL, err := r.CreatePR(wtDir, title, body, branch, r.Base)
	if err != nil {
		logger.Error("cli/pr", "gh pr create failed: %v", err)
		return fmt.Errorf("branch %q pushed, but gh pr create failed: %w", branch, err)
	}

	logger.Info("cli/pr", "created pull request %s for task %s (base=%s)", prURL, taskID, r.Base)
	fmt.Fprintf(ctx.Out, "Pull Request created: %s\n", prURL)
	return nil
}
