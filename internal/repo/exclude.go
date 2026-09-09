package repo

// Keeping Orbit's own files out of the branch a task hands back.
//
// Orbit writes a copy of every decision under .orbit/decisions/ in the
// worktree, because a decision belongs where the work happened. What it must
// never become is part of the user's commit: CommitWorktree stages the whole
// tree, an engine runs git of its own besides, and the copies rode into a
// pull request nobody put them in — reported from a real board on
// 2026-09-09, five decision files staged beside the work in two tasks at
// once.
//
// The line goes in git's info/exclude and not in a .gitignore. A .gitignore
// is the project's file, and this is not the project's business: Orbit reads
// the repository that is there rather than asking for one that suits it.
// info/exclude is git's own place for "this checkout, not this project", it
// is never committed, and it is read by every worktree — git keeps info/ in
// the common directory, so a copy written under a linked worktree's own git
// directory is not read at all.
//
// A path git already tracks is unaffected by an exclude, so a repository
// that keeps a .orbit/ of its own in the tree carries on committing it.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// orbitDir is the directory Orbit keeps its own files in inside a checkout.
// task.OrbitDir is the same name said where the decisions are written; this
// one is what git is told to ignore.
const orbitDir = ".orbit"

// excludeLine is what is written, with the slash that says "a directory".
const excludeLine = orbitDir + "/"

// excludeOrbit tells the repository holding dir to ignore Orbit's own
// directory, and says so once however many times it is called.
//
// dir may be the repository or any worktree of it: the file written is the
// one they share.
func excludeOrbit(dir string) error {
	common, err := git(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return fmt.Errorf("find the git directory of %q: %w", dir, err)
	}

	// Relative to the directory git was asked from, when git answers
	// relatively, which it does for a repository the caller is standing in.
	gitDir := strings.TrimSpace(common)
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(dir, gitDir)
	}

	path := filepath.Join(gitDir, "info", "exclude")

	said, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %q: %w", path, err)
	}

	if excluded(string(said)) {
		return nil
	}

	if err := mkdir(filepath.Dir(path)); err != nil {
		return err
	}

	return writeExclude(path, string(said))
}

// excluded says whether the line is already in a file's text.
func excluded(said string) bool {
	for _, line := range strings.Split(said, "\n") {
		if strings.TrimSpace(line) == excludeLine {
			return true
		}
	}

	return false
}

// writeExclude puts the line at the end of what is already there, under a comment
// saying who wrote it: a person reading this file later is owed the reason a
// line they did not write is in it.
//
// The newline before is written when what is there does not end in one,
// because a file whose last line has no newline would otherwise be joined to
// the comment.
func writeExclude(path, said string) error {
	add := "# Orbit's own copies of its decisions, which are not this\n" +
		"# project's files. The record they are copied from is the truth.\n" +
		excludeLine + "\n"

	if said != "" && !strings.HasSuffix(said, "\n") {
		add = "\n" + add
	}

	// 0644 because info/exclude is git's own file, written that way by git
	// itself when it puts the default one there.
	if err := os.WriteFile(path, []byte(said+add), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("write %q: %w", path, err)
	}

	return nil
}

// unstageOrbit takes Orbit's own files out of the index of a worktree that
// staged them before the exclude was there.
//
// Only the files git has as added and does not have in HEAD, one path at a
// time. `git reset -- .orbit` would be the short way to write it and the
// wrong one: on a repository that tracks a .orbit/ of its own it would throw
// away a change the task really made to a file that really belongs to the
// project.
func unstageOrbit(wtDir string) error {
	out, err := git(wtDir, "diff", "--cached", "--name-only", "--diff-filter=A", "--", orbitDir)
	if err != nil {
		return fmt.Errorf("look for staged Orbit files in %q: %w", wtDir, err)
	}

	for _, path := range strings.Fields(out) {
		if _, err := git(wtDir, "reset", "-q", "--", path); err != nil {
			return fmt.Errorf("unstage %q in %q: %w", path, wtDir, err)
		}
	}

	return nil
}
