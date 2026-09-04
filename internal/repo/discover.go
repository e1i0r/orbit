package repo

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/e1i0r/orbit/internal/store"
)

// Discover walks a root and returns every repository beneath it.
//
// It does not descend into a repository once it finds one: a checkout sitting
// inside another checkout is a vendored copy or a fixture, not a second
// project you meant to work on.
//
// Orbit's own state root is skipped by path. Every task's worktree is a real
// checkout living under it, and offering a user their own throwaway
// worktrees as repositories to start further tasks against is nonsense. The
// dotted-directory rule below is not enough for this: it only worked while
// the root happened to be called ~/.orbit, and $ORBIT_HOME can be pointed at
// any name at all.
//
// Symlinks are not followed, and that is deliberate rather than an
// oversight. A link that points at its own ancestor turns the walk into a
// hang, and a repository reachable by two paths is filed under two different
// store hashes — two records for one project, which is worse than not
// seeing it twice.
func Discover(root string) ([]Repo, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", root, err)
	}

	state, stateErr := store.RootPath()
	if stateErr != nil {
		// A home directory that cannot be resolved is not a reason to
		// refuse to list repositories; there is simply nothing to skip.
		state = ""
	}

	var found []Repo

	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}

			return err
		}

		if !d.IsDir() {
			return nil
		}

		if state != "" && path == state {
			return filepath.SkipDir
		}

		if name := d.Name(); path != abs && (strings.HasPrefix(name, ".") || isIgnoredDir(name)) {
			return filepath.SkipDir
		}

		if _, statErr := os.Stat(filepath.Join(path, ".git")); statErr != nil {
			// Not a repository, so there is nothing here to report. statErr is a
			// probe, not a failure: walking on is the whole point.
			return nil //nolint:nilerr // absence of .git is the ordinary case
		}

		r, openErr := Open(path)
		if openErr != nil {
			// A directory has a .git entry but Open failed — it looks like a
			// repository but cannot be opened, perhaps because of git's dubious
			// ownership safety check when the repository's owner differs from the
			// running user. Omit it from results instead of failing the whole walk,
			// because one unopenable directory must not blank the entire listing.
			// Descending into it is refused: it is still a repository, and its
			// contents are not separate projects to start tasks against.
			return filepath.SkipDir //nolint:nilerr // deliberate: skip this one, keep the listing
		}

		found = append(found, r)

		return filepath.SkipDir
	})
	if err != nil {
		return nil, fmt.Errorf("walk %q: %w", abs, err)
	}

	sort.Slice(found, func(i, j int) bool { return found[i].Path < found[j].Path })

	return found, nil
}

// isIgnoredDir reports directory names that cannot be repositories to track
// tasks against, such as package manager caches, dependency folders, and build
// outputs. Skipping them avoids traversing tens of thousands of nested folders
// and exhausting system file descriptors.
func isIgnoredDir(name string) bool {
	switch name {
	case "node_modules", "vendor", "__pycache__", "build", "dist", "target", "coverage":
		return true
	default:
		return false
	}
}
