package repo

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Repo is one repository and the three facts Orbit needs about it.
//
// Remote is a field rather than the constant "origin" because it is not
// always origin — the repository this tool was built against calls its remote
// "acme", and the previous version of this program hardcoded "origin" in
// ten files and could not run against it.
type Repo struct {
	Path   string
	Name   string
	Remote string

	// Base is the branch a task's worktree is cut from, and is empty when
	// the repository is not on a branch at all. `rev-parse --abbrev-ref
	// HEAD` answers the literal string "HEAD" on a detached checkout, and
	// carrying that around would mean displaying a branch named HEAD and
	// then trying to cut a worktree from it. Empty is the honest answer:
	// there is no branch here.
	Base string
}

// Open reads the shape of the repository at a path.
func Open(path string) (Repo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Repo{}, fmt.Errorf("resolve %q: %w", path, err)
	}
	top, err := git(abs, "rev-parse", "--show-toplevel")
	if err != nil {
		return Repo{}, fmt.Errorf("%q is not a repository: %w", abs, err)
	}
	remote, err := pickRemote(top)
	if err != nil {
		return Repo{}, err
	}
	base, err := git(top, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return Repo{}, fmt.Errorf("read the current branch of %q: %w", top, err)
	}
	if base == "HEAD" {
		// Detached: git has no branch name to give, so neither have we.
		// The repository is still worth listing — it is only starting a
		// task against it that cannot work, and AddWorktree says so.
		base = ""
	}
	return Repo{
		Path:   top,
		Name:   filepath.Base(top),
		Remote: remote,
		Base:   base,
	}, nil
}

// pickRemote returns origin when it exists, the only remote when there is
// exactly one, and the first otherwise. No remote at all is a valid answer:
// a local repository is still a repository.
func pickRemote(dir string) (string, error) {
	out, err := git(dir, "remote")
	if err != nil {
		return "", fmt.Errorf("list the remotes of %q: %w", dir, err)
	}
	if out == "" {
		return "", nil
	}
	names := splitLines(out)
	for _, n := range names {
		if n == "origin" {
			return "origin", nil
		}
	}
	return names[0], nil
}

// splitLines turns git's output into a slice, dropping the blanks.
func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}
