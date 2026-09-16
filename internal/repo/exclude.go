package repo

// Keeping Orbit's own bookkeeping out of the branch a task hands back.
//
// Orbit writes a copy of every decision under .orbit/decisions/ in the
// worktree, because a decision belongs where the work happened. What it must
// never become is part of the user's commit: CommitWorktree stages the whole
// tree, an engine runs git of its own besides, and the copies rode into a
// pull request nobody put them in — reported from a real board on
// 2026-09-09, five decision files staged beside the work in two tasks at
// once.
//
// What is excluded is that directory and not .orbit/ around it, and the
// difference is the whole of what a repository-scoped rule is for. Two
// different things live under .orbit/: the decisions, which are Orbit's
// bookkeeping and never the project's, and .orbit/knowledge/, which is where
// a rule goes when somebody chooses this checkout for it — the window offers
// that choice with the words "travels with it, so whoever clones the project
// gets it".
//
// Excluding .orbit/ whole made that sentence false. A rule filed against a
// repository never left the machine it was written on, so "this checkout"
// and "this machine" were the same answer with two names, and the form was
// asking a question that changed nothing.
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

// orbitDir is the directory Orbit keeps its own files in inside a checkout,
// and decisionsDir the one under it that is nobody's but Orbit's.
// task.OrbitDir is the same name said where the decisions are written; these
// are what git is told about.
const (
	orbitDir     = ".orbit"
	decisionsDir = orbitDir + "/decisions"
)

// excludeLine is what is written, with the slash that says "a directory".
const excludeLine = decisionsDir + "/"

// wasExcludeLine is what older Orbits wrote: the whole of .orbit/, which
// took the knowledge with it.
//
// It is named here because it has to be taken out rather than left alone. A
// broader line already in the file goes on winning however narrow the new
// one is, so a checkout Orbit has already touched would keep the old
// behaviour for ever and the fix would only reach machines that had never
// run it.
const wasExcludeLine = orbitDir + "/"

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

	if err := mkdir(filepath.Dir(path)); err != nil {
		return err
	}

	// A file an older Orbit wrote the broad line into is corrected in place
	// rather than appended to.
	if narrowed, was := narrow(string(said)); was {
		return put(path, narrowed)
	}

	if excluded(string(said)) {
		return nil
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

// narrow rewrites the broad line an older Orbit wrote into the one that only
// covers the decisions, and says whether it found one.
//
// Only that exact line, and it is not removed but replaced: somebody who put
// ".orbit/" in this file by hand meant it, and the comment above Orbit's own
// line is what says which is which. Rewriting in place rather than appending
// keeps that comment pointing at the line it explains.
func narrow(said string) (string, bool) {
	lines := strings.Split(said, "\n")

	found := false

	for i, line := range lines {
		if strings.TrimSpace(line) == wasExcludeLine {
			lines[i], found = excludeLine, true
		}
	}

	if !found {
		return said, false
	}

	return strings.Join(lines, "\n"), true
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
		"# Rules under .orbit/knowledge/ are not excluded: those were filed\n" +
		"# against this checkout on purpose, to travel with it.\n" +
		excludeLine + "\n"

	if said != "" && !strings.HasSuffix(said, "\n") {
		add = "\n" + add
	}

	return put(path, said+add)
}

// put writes the file back.
//
// 0644 because info/exclude is git's own file, written that way by git
// itself when it puts the default one there.
func put(path, said string) error {
	if err := os.WriteFile(path, []byte(said), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("write %q: %w", path, err)
	}

	return nil
}

// unstageOrbit takes Orbit's own decision copies out of the index of a
// worktree that staged them before the exclude was there.
//
// The decisions and not the whole of .orbit/, for the reason the exclude is
// narrow: a rule written under .orbit/knowledge/ during a run belongs to the
// checkout it was filed against, and taking it back out of the index is
// Orbit undoing what somebody asked for.
//
// Only the files git has as added and does not have in HEAD, one path at a
// time. `git reset -- .orbit/decisions` would be the short way to write it
// and the wrong one: on a repository that tracks a .orbit/ of its own it
// would throw away a change the task really made to a file that really
// belongs to the project.
func unstageOrbit(wtDir string) error {
	out, err := git(wtDir, "diff", "--cached", "--name-only", "--diff-filter=A", "--", decisionsDir)
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
