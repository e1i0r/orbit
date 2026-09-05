package repo

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/e1i0r/orbit/internal/store"
)

// A Found is a repository the walk saw: where it is and what it is called,
// and nothing else. Everything else a Repo knows — its remote, the branch it
// stands on — costs a git subprocess to learn, and the caller that asks
// twice a second does not read any of it.
type Found struct {
	Path string
	Name string
}

// Paths walks a root and returns every repository beneath it, without
// opening any of them.
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
// seeing it twice. The path that is reported is the resolved one all the
// same, because that is the path git answers with and the one the record is
// keyed by.
func Paths(root string) ([]Found, error) {
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

	var found []Found

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

		dotGit := filepath.Join(path, ".git")

		info, statErr := os.Stat(dotGit)

		switch {
		case statErr != nil:
			return dotGitVerdict(statErr)
		case info.IsDir(), gitfilePointsAtARepository(dotGit):
			found = append(found, foundAt(path))
		}
		// A .git that is neither of those looks like a repository and is
		// not one. It is left out of the listing, and not descended into
		// either: whatever is inside it is not a set of separate projects.
		return filepath.SkipDir
	})
	if err != nil {
		return nil, fmt.Errorf("walk %q: %w", abs, err)
	}

	sort.Slice(found, func(i, j int) bool { return found[i].Path < found[j].Path })

	return found, nil
}

// dotGitVerdict turns the failure of the .git probe into what the walk does
// next. It is only ever handed an error: a probe that succeeded found a
// repository, and that is the caller's own case.
//
// The three answers are three different facts, and the middle one used to be
// spelled the same as the first. A directory with no .git in it is the
// ordinary case and the walk continues into it. A directory that cannot be
// read is left alone, because whether it is a repository is not knowable and
// descending into it would be a guess. Anything else — and the one that
// matters is the machine having run out of file descriptors — stops the walk
// and says why: under that error every probe fails, so a walk that treated
// it as "not a repository" would stop recognising repositories exactly when
// it started descending into all of their contents, turning a shortage into
// a storm.
func dotGitVerdict(err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case errors.Is(err, fs.ErrPermission):
		return filepath.SkipDir
	default:
		return err
	}
}

// gitdirPrefix is how git's own pointer file begins.
const gitdirPrefix = "gitdir:"

// gitfilePointsAtARepository is whether a .git that is a file is git's
// pointer to a repository kept elsewhere — which is what a worktree and a
// submodule have — rather than a file that happens to carry the name.
//
// Reading the first seven bytes is what replaces running git here. It is the
// one case where the walk cannot tell from the directory entry alone, and it
// costs an open and a read of a file that is one line long.
func gitfilePointsAtARepository(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}

	defer func() { _ = f.Close() }()

	head := make([]byte, len(gitdirPrefix))
	if _, err := io.ReadFull(f, head); err != nil {
		return false
	}

	return string(head) == gitdirPrefix
}

// foundAt names a repository by the path git would answer with. Resolving is
// two syscalls and no subprocess, and a path that cannot be resolved is
// reported as it was walked rather than dropped.
func foundAt(path string) Found {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		real = path
	}

	return Found{Path: real, Name: filepath.Base(real)}
}

// Discover is Paths with every repository opened, so that each one carries
// its remote and the branch it stands on.
//
// Opening is three git subprocesses apiece, which is why it is a separate
// door from the walk: this is what `orbit repos` prints and what a single
// command pays once, and the board's own enumeration asks Paths instead.
func Discover(root string) ([]Repo, error) {
	paths, err := Paths(root)
	if err != nil {
		return nil, err
	}

	found := make([]Repo, 0, len(paths))

	for _, p := range paths {
		r, openErr := Open(p.Path)
		if openErr != nil {
			// A directory has a .git entry but Open failed — it looks like a
			// repository but cannot be opened, perhaps because of git's dubious
			// ownership safety check when the repository's owner differs from the
			// running user. Omit it from results instead of failing the whole walk,
			// because one unopenable directory must not blank the entire listing.
			continue
		}

		found = append(found, r)
	}

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
