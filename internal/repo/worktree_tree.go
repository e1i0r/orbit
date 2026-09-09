package repo

// What a checkout holds, by path.
//
// It is git's answer and not a walk of the directory: git already knows
// which files are the repository's and which are build output, caches and
// somebody's editor state, and it knows it from the .gitignore the project
// wrote rather than from a list of exclusions Orbit would have to keep. A
// walk would put node_modules on the map.

import "strings"

// WorktreeFiles is every file git tracks in the worktree, by its path from
// the worktree's root, in git's own order.
//
// Untracked files are not here, and that is the right answer for a map: a
// file nobody has added yet is not part of the repository, and a map that
// showed it would show a different shape to every reader depending on what
// their editor happened to have written.
func (r Repo) WorktreeFiles(wtDir string) ([]string, error) {
	out, err := git(wtDir, "ls-files", "-z")
	if err != nil {
		return nil, err
	}

	// -z, because a path may hold anything a filesystem allows including a
	// newline, and git quotes such a path when it separates them with one.
	// A quoted path is a path this would have to unquote, and unquoting is
	// where a map starts showing files that are not there.
	var files []string

	for _, path := range strings.Split(out, "\x00") {
		if path != "" {
			files = append(files, path)
		}
	}

	return files, nil
}
