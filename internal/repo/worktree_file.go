package repo

// One file of a worktree, read as it stands.
//
// It is here and not beside the diff because it is a different question. The
// diff is what git says changed; this is what the checkout holds, and the
// reader who wants the lines above a hunk is asking the second question with
// the first one on screen.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WorktreeFile is one file as it stands in the worktree, by its path from
// the worktree's root.
//
// It is what a reader needs to open a diff out: git writes three lines of
// context around each hunk, and the question "what is above this" is
// answered by the file itself rather than by a second, wider diff.
//
// The path is refused rather than cleaned when it leaves the worktree. A
// caller that meant ../../etc/passwd is not a caller with a typo, and a
// server that quietly answers the file it decided they meant is a server
// nobody can reason about.
func (r Repo) WorktreeFile(wtDir, path string) (string, error) {
	inside, err := under(wtDir, path)
	if err != nil {
		return "", err
	}

	body, err := os.ReadFile(inside)
	if err != nil {
		return "", fmt.Errorf("read %q: %w", path, err)
	}

	return string(body), nil
}

// under is where a path inside a worktree actually is, and an error for one
// that is not inside it.
func under(wtDir, path string) (string, error) {
	root, err := filepath.Abs(wtDir)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", wtDir, err)
	}

	full, err := filepath.Abs(filepath.Join(root, filepath.Clean("/"+path)))
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", path, err)
	}

	if full != root && !strings.HasPrefix(full, root+string(filepath.Separator)) {
		return "", fmt.Errorf("%q is not inside the worktree", path)
	}

	return full, nil
}
