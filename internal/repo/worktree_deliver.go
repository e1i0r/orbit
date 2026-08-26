package repo

import (
	"fmt"
	"os/exec"
	"strings"
)

// CommitWorktree stages and commits all untracked and modified files in the worktree.
func (r Repo) CommitWorktree(wtDir, message string) error {
	if _, err := git(wtDir, "add", "-A"); err != nil {
		return fmt.Errorf("stage changes in %q: %w", wtDir, err)
	}
	status, err := git(wtDir, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("check status in %q: %w", wtDir, err)
	}
	if strings.TrimSpace(status) == "" {
		return nil // nothing to commit
	}
	if _, err := git(wtDir, "commit", "-m", message); err != nil {
		return fmt.Errorf("commit in %q: %w", wtDir, err)
	}
	return nil
}

// PushBranch pushes the task branch from the worktree to the remote origin.
func (r Repo) PushBranch(wtDir, branch string) error {
	if _, err := git(wtDir, "push", "-u", "origin", "HEAD:"+branch); err != nil {
		return fmt.Errorf("push branch %q to origin: %w", branch, err)
	}
	return nil
}

// CreatePR invokes the GitHub CLI `gh pr create` inside the worktree directory.
func (r Repo) CreatePR(wtDir, title, body, headBranch, baseBranch string) (string, error) {
	args := []string{"pr", "create", "--head", headBranch, "--title", title, "--body", body}
	if baseBranch != "" {
		args = append(args, "--base", baseBranch)
	}
	cmd := exec.Command("gh", args...)
	cmd.Dir = wtDir
	out, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("gh pr create: %s: %w", outputStr, err)
	}
	return outputStr, nil
}
