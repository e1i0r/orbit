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

// SyncBaseBranch fetches and merges the latest base branch into the task worktree.
func (r Repo) SyncBaseBranch(wtDir, baseBranch string) error {
	if baseBranch == "" {
		return nil
	}
	if out, err := git(wtDir, "fetch", "origin", baseBranch); err != nil {
		return fmt.Errorf("fetch origin %q: %s: %w", baseBranch, out, err)
	}
	if out, err := git(wtDir, "merge", "--no-edit", "origin/"+baseBranch); err != nil {
		return fmt.Errorf("sync base branch %q into worktree: %s: %w", baseBranch, out, err)
	}
	return nil
}

// CreatePR invokes the GitHub CLI `gh pr create` inside the worktree directory.
// If a PR already exists for the head branch, it fetches and returns its URL.
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
		if strings.Contains(outputStr, "already exists") {
			viewCmd := exec.Command("gh", "pr", "view", headBranch, "--json", "url", "-q", ".url")
			viewCmd.Dir = wtDir
			if viewOut, viewErr := viewCmd.CombinedOutput(); viewErr == nil {
				return strings.TrimSpace(string(viewOut)), nil
			}
			return outputStr, nil
		}
		return "", fmt.Errorf("gh pr create: %s: %w", outputStr, err)
	}
	return outputStr, nil
}

// MergePR merges the GitHub Pull Request for the task using the GitHub CLI.
func (r Repo) MergePR(wtDir, branch string) (string, error) {
	cmd := exec.Command("gh", "pr", "merge", branch, "--squash", "--delete-branch")
	cmd.Dir = wtDir
	out, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("gh pr merge: %s: %w", outputStr, err)
	}
	return outputStr, nil
}

// ClosePR closes the GitHub Pull Request for the task using the GitHub CLI.
func (r Repo) ClosePR(wtDir, branch string) (string, error) {
	cmd := exec.Command("gh", "pr", "close", branch, "--comment", "Closed from Orbit.")
	cmd.Dir = wtDir
	out, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("gh pr close: %s: %w", outputStr, err)
	}
	return outputStr, nil
}
