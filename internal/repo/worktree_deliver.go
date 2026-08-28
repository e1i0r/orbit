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

// PushBranch pushes the task branch from the worktree to the repository's
// remote.
//
// The remote is r.Remote and not the constant "origin", which is what these
// two functions used to say. Repo carries the name for one reason — it is
// not always origin, and the repository this tool was built against calls
// its remote "acme" — and a delivery that hardcodes the word fails on such a
// repository at the last step of the flow, after the work is committed.
//
// A repository with no remote at all is refused here rather than handed to
// git as an empty argument, because `git push -u "" HEAD:branch` fails with
// a message about a repository named "" and nothing about what Orbit was
// trying to do.
func (r Repo) PushBranch(wtDir, branch string) error {
	if r.Remote == "" {
		return fmt.Errorf("push branch %q: %q has no remote to push to", branch, r.Name)
	}
	if _, err := git(wtDir, "push", "-u", r.Remote, "HEAD:"+branch); err != nil {
		return fmt.Errorf("push branch %q to %s: %w", branch, r.Remote, err)
	}
	return nil
}

// SyncBaseBranch fetches and merges the latest base branch into the task
// worktree.
//
// No remote means there is nothing newer to sync from, which is the same
// answer as no base branch: the delivery that follows will stop at
// PushBranch, and it will say so in terms of the push rather than of a fetch
// nobody asked about.
//
// git's own stderr is not interpolated into these messages any more. It
// used to be — `fetch origin %q: %s: %w` with the output as %s — and it was
// always empty, because git() returns "" on failure and folds stderr into
// the error it returns. The message read "fetch origin "main": : exit
// status 1", and the colon with nothing in front of it was the part a
// reader would have gone looking for.
func (r Repo) SyncBaseBranch(wtDir, baseBranch string) error {
	if baseBranch == "" || r.Remote == "" {
		return nil
	}
	if _, err := git(wtDir, "fetch", r.Remote, baseBranch); err != nil {
		return fmt.Errorf("fetch %s %q: %w", r.Remote, baseBranch, err)
	}
	if _, err := git(wtDir, "merge", "--no-edit", r.Remote+"/"+baseBranch); err != nil {
		return fmt.Errorf("sync base branch %q into worktree: %w", baseBranch, err)
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
			viewOut, viewErr := viewCmd.CombinedOutput()
			if viewErr != nil {
				// The old answer here was the text gh had printed while
				// failing, returned as the URL with a nil error. The caller
				// prints what it is given as "Pull Request created: %s", so
				// a reader was told a pull request had been created and
				// handed an error message to click on.
				return "", fmt.Errorf("a pull request for %q already exists, but gh pr view could not say where: %s: %w",
					headBranch, strings.TrimSpace(string(viewOut)), viewErr)
			}
			return strings.TrimSpace(string(viewOut)), nil
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
