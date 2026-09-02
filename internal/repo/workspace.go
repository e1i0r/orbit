package repo

// The set of repositories a task is allowed to reach into.

import (
	"os"
	"path/filepath"
)

// WorkspaceEnv is where a running phase is told which workspace it is in.
//
// A phase runs in a worktree under the state root, which is nowhere near the
// checkouts it may join, so the child cannot work the answer out from its
// own directory. The run puts it here and `orbit join` reads it; a reader
// typing that command by hand has neither, and Workspace below is the
// fallback for them.
const WorkspaceEnv = "ORBIT_WORKSPACE"

// Workspace is the directory the repositories of a task are looked for in.
//
// It is the parent of the repository the task was written against, which is
// a guess about a layout and is worth stating as one: projects that belong
// to the same piece of work are usually checked out side by side —
// api/, app/, scripts/ under one directory — and that is the arrangement
// this answer is right for. It is a guess and not a setting because the
// alternative is asking every user to declare the thing §5 of the many-repos
// spec exists to stop asking about, and because $ORBIT_WORKSPACE overrides
// it for a layout that is shaped otherwise.
//
// The walk that follows this can be expensive when the guess is wide — a
// repository cloned straight into $HOME makes the whole home directory the
// workspace. Discover does not descend into a repository once it finds one
// and skips dotted directories, so the cost is the breadth of the tree
// rather than its depth, and it is paid once per run.
//
// A task with no repository at all has no parent to take, and the answer is
// nothing rather than the current directory. filepath.Dir("") is ".", and
// "." in a phase of such a task is a worktree under the state root — a
// workspace nobody chose, holding no repositories, walked on every run.
func Workspace(r Repo) string {
	if set := os.Getenv(WorkspaceEnv); set != "" {
		return set
	}

	if r.Path == "" {
		return ""
	}

	return filepath.Dir(r.Path)
}

// Siblings is every repository of a workspace, the one the task already
// works in included.
//
// It is the candidate set and nothing more: which of them a task ends up
// reaching into is decided by the work, one worktree at a time, and is never
// read off this list.
func Siblings(r Repo) ([]Repo, error) {
	where := Workspace(r)
	if where == "" {
		return nil, nil
	}

	return Discover(where)
}
