package repo

import (
	"fmt"
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
// The remote is r.Remote and not the constant "origin". Repo carries the
// name for one reason — it is not always origin, and the repository this
// tool was built against calls its remote "acme" — and a delivery that
// hardcodes the word fails on such a repository at the last step of the
// flow, after the work is committed.
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
// git's own stderr is not interpolated into these messages. Written as
// `fetch origin %q: %s: %w` with the output as %s it is always empty,
// because git() returns "" on failure and folds stderr into the error it
// returns: the message reads "fetch origin "main": : exit status 1", and
// the colon with nothing in front of it is the part a reader would have
// gone looking for.
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

// CreatePR opens a pull request for the task branch and answers where it is.
//
// gh writes the URL to stdout and everything else it has to say — an
// upgrade notice, a warning about the default branch — to stderr, so the
// answer is stdout and nothing else. Both streams together is a link with a
// paragraph in front of it.
func (r Repo) CreatePR(wtDir, title, body, headBranch, baseBranch string) (string, error) {
	args := []string{"pr", "create", "--head", headBranch, "--title", title, "--body", body}
	if baseBranch != "" {
		args = append(args, "--base", baseBranch)
	}

	url, err := gh(wtDir, args...)
	if err == nil {
		return url, nil
	}

	if !strings.Contains(err.Error(), "already exists") {
		return "", err
	}

	// A pull request for this branch is already open, which is the ordinary
	// answer to delivering a task twice. gh says so and does not say where,
	// so it is asked.
	url, viewErr := gh(wtDir, "pr", "view", headBranch, "--json", "url", "-q", ".url")
	if viewErr != nil {
		// The old answer here was the text gh had printed while failing,
		// returned as the URL with a nil error. The caller prints what it is
		// given as "Pull Request created: %s", so a reader was told a pull
		// request had been created and handed an error message to click on.
		return "", fmt.Errorf("a pull request for %q already exists, but gh pr view could not say where: %w", headBranch, viewErr)
	}

	return url, nil
}

// MergePR squashes and merges the task's pull request, and deletes the
// branch behind it.
//
// It answers an error and nothing else. gh's confirmation is written to
// stderr, not stdout — it is a status line for a human at a terminal rather
// than an answer — and a caller that printed it was printing whatever
// happened to be on that stream. What a reader needs to be told is that the
// merge happened, and the caller knows which task it was for.
func (r Repo) MergePR(wtDir, branch string) error {
	_, err := gh(wtDir, "pr", "merge", branch, "--squash", "--delete-branch")

	return err
}

// ClosePR closes the task's pull request, with the comment the caller gave.
//
// The comment is a parameter because this package runs git and gh and does
// not write English: the sentence a reader leaves on somebody else's
// repository belongs where the rest of the words do. An empty comment closes
// the pull request without leaving one.
func (r Repo) ClosePR(wtDir, branch, comment string) error {
	args := []string{"pr", "close", branch}
	if comment != "" {
		args = append(args, "--comment", comment)
	}

	_, err := gh(wtDir, args...)

	return err
}
