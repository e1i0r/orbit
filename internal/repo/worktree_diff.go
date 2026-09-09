package repo

// What a task's worktree has changed, counted rather than shown.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Change is one file a worktree has touched, and how much of it.
//
// Added and Deleted are lines, as git counts them, and both are -1 for a
// binary file — git prints "-" for those, and a zero would read as a file
// nothing happened to. A caller totalling lines has to decide what a binary
// file is worth; this says which one it is looking at rather than deciding
// for it.
type Change struct {
	Path    string
	Added   int
	Deleted int
}

// Binary reports whether git counted no lines because there are none to
// count.
func (c Change) Binary() bool { return c.Added < 0 || c.Deleted < 0 }

// Lines is what this file contributes to a diff budget: everything written
// and everything removed.
//
// A binary file counts as nothing rather than as one, because the budget is
// about how much there is to read and a changed PNG is not a line of code.
// It is still in the list, so a scope check can refuse a file that had no
// business changing whatever its type.
func (c Change) Lines() int {
	if c.Binary() {
		return 0
	}

	return c.Added + c.Deleted
}

// WorktreeChanges is every file the task's worktree has changed against the
// branch it was cut from.
//
// Committed and uncommitted alike: a phase that committed its work and a
// phase that left it in the tree have both changed the same amount, and a
// budget that counted only one of them would be a budget an engine could
// walk past by committing.
//
// Untracked files are marked with intent-to-add first, for the same reason
// the window's diff does it: a new file is the largest change there is, and
// a count that left it out would pass the biggest diffs of all.
func (r Repo) WorktreeChanges(wtDir string) ([]Change, error) {
	if _, err := git(wtDir, "add", "-N", "--ignore-errors", "."); err != nil {
		// Not a failure of the count. A worktree with nothing to add
		// answers non-zero on some versions, and a file git refuses to
		// stage is one the diff below simply will not mention.
		_ = err //nolint:wsl // deliberate: the error is the answer, not a fault
	}

	// --no-renames because git detects them by default and writes the pair
	// as one path, `f => src/renamed.go`. Everything downstream reads the
	// third column as a filename: path.Base of that is `pkg.json`, the
	// `.orbit/` prefix test misses a file moved out of it, and coChanged
	// looks for it in `git log --name-only`, which never spells a path that
	// way. Counted as two paths, added and deleted, they are all right.
	base := r.against(wtDir)
	args := append([]string{"diff", "--numstat", "--no-renames"}, base...)

	out, err := git(wtDir, args...)
	if err != nil {
		// Only when there was no base to compare against, and not for any
		// other failure. A merge-base diff that broke on unrelated
		// histories or a shallow clone fell back to a diff of the working
		// tree alone, which counts none of the committed lines: the budget
		// then read a small number that looked right and was not.
		if len(base) > 0 {
			return nil, fmt.Errorf("count what %q changed: %w", wtDir, err)
		}

		// Without a base there is nothing to compare against and the
		// working tree is the whole answer — a worktree cut before its base
		// existed, or one whose base has been deleted since, still has
		// changes worth counting.
		out, err = git(wtDir, "diff", "--numstat", "--no-renames")
		if err != nil {
			return nil, fmt.Errorf("count what %q changed: %w", wtDir, err)
		}
	}

	return numstat(out), nil
}

// WorktreeDiff is what the task changed, as git writes it.
//
// The text and not the counts: a reader looking at the change wants the
// hunks. WorktreeChanges answers the same question in numbers, for the gates
// that only need a size.
//
// Against the branch the worktree was cut from, and the working tree alone
// when there is none — the same rule the counts follow, for the same reason:
// a worktree cut before its base existed still has changes worth reading.
//
// internal/ui has its own copy of this shaped for a window that redraws
// every half second, with a pending state and a deadline of its own. The two
// should end up as one; this one exists because a reader outside the
// terminal needs the same answer and may not reach into internal/ui.
func (r Repo) WorktreeDiff(wtDir string, how DiffOptions) (string, error) {
	// Untracked files are marked intent-to-add so the diff mentions them. A
	// file the agent wrote and never staged is the most interesting file
	// there is, and it is invisible to git diff without this.
	if _, err := git(wtDir, "add", "-N", "--ignore-errors", "."); err != nil {
		_ = err //nolint:wsl // a worktree with nothing to add answers non-zero on some versions
	}

	base := r.against(wtDir)

	args := append([]string{"diff"}, how.args()...)

	out, err := git(wtDir, append(args, base...)...)
	if err == nil {
		return out, nil
	}

	if len(base) > 0 {
		return "", fmt.Errorf("read what %q changed: %w", wtDir, err)
	}

	out, err = git(wtDir, args...)
	if err != nil {
		return "", fmt.Errorf("read what %q changed: %w", wtDir, err)
	}

	return out, nil
}

// DiffOptions is how a reader asked for the diff.
//
// The zero value is git's own default, which is what every caller before
// this wanted: three lines of context, and whitespace counted.
type DiffOptions struct {
	// IgnoreWhitespace leaves out the lines that differ only in spacing. A
	// reformatting run that touched two hundred files and changed nothing
	// buries the one line somebody has to read, and this is the only way to
	// see past it.
	IgnoreWhitespace bool
}

// args is the options as git spells them.
func (d DiffOptions) args() []string {
	if d.IgnoreWhitespace {
		return []string{"-w"}
	}

	return nil
}

// WorktreeAddedLines is every line the task added to one file, without the
// leading plus.
//
// Added and not changed: what a dependency gate asks is what appeared, and a
// line that was removed cannot have brought a library with it. -U0 so that
// the context lines around a change — which are somebody else's
// dependencies, already there — are not read as new ones.
func (r Repo) WorktreeAddedLines(wtDir, path string) ([]string, error) {
	base := r.against(wtDir)
	args := append([]string{"diff", "-U0"}, base...)
	args = append(args, "--", path)

	out, err := git(wtDir, args...)
	if err != nil {
		// As in WorktreeChanges: a base that was given and would not diff is
		// a failure, not a reason to answer with the working tree — which
		// carries none of the committed lines, and let the dependency gate
		// approve a library the task had already committed.
		if len(base) > 0 {
			return nil, fmt.Errorf("read what %q added to %q: %w", wtDir, path, err)
		}

		out, err = git(wtDir, "diff", "-U0", "--", path)
		if err != nil {
			return nil, fmt.Errorf("read what %q added to %q: %w", wtDir, path, err)
		}
	}

	var added []string

	for _, line := range splitLines(out) {
		// +++ is the header naming the file, not a line of it.
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added = append(added, strings.TrimPrefix(line, "+"))
		}
	}

	return added, nil
}

// against is what the count is taken against: the branch the worktree was
// cut from, through its merge base so that work somebody else pushed to that
// branch since is not read as this task's.
//
// The remote-tracking branch when there is one, for the reason WorktreeAhead
// names — a local base nobody has fetched in a week is behind its remote,
// and counting against it reports somebody else's work as this task's.
//
// Two arguments and not one: `--merge-base=X` is not a form git parses, and
// git takes it for a path, answers nothing, and leaves the caller reading a
// diff of the working tree alone — every committed line of the task missing,
// silently.
func (r Repo) against(wtDir string) []string {
	base := r.cutFrom(wtDir)
	if base == "" {
		return nil
	}

	if tracking := r.Remote + "/" + base; r.Remote != "" && r.resolves(wtDir, tracking) {
		return []string{"--merge-base", tracking}
	}

	return []string{"--merge-base", base}
}

// cutFrom is the branch this worktree was cut from.
//
// Read off the branch, where AddWorktree wrote it, and not off r.Base. r.Base
// is whichever branch the repository's own checkout is standing on right now,
// which is a different question and a different answer the moment somebody
// switches branches or detaches HEAD while a task is running — at which point
// the count was taken against a branch the task never left, or against
// nothing at all.
//
// A worktree from before this was recorded falls back to r.Base, which is
// what it was measured against for its whole life.
func (r Repo) cutFrom(wtDir string) string {
	gitDir, err := git(wtDir, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return r.Base
	}

	body, err := os.ReadFile(filepath.Join(strings.TrimSpace(gitDir), baseFile))
	if err != nil || strings.TrimSpace(string(body)) == "" {
		return r.Base
	}

	return strings.TrimSpace(string(body))
}

// numstat reads git's own three columns: added, deleted, path.
//
// A line it cannot read is skipped rather than guessed at. numstat's shape
// is git's to change, and a path with a tab in it would otherwise be counted
// as a file called something else.
func numstat(out string) []Change {
	var changes []Change

	for _, line := range splitLines(out) {
		cols := strings.SplitN(line, "\t", 3)
		if len(cols) != 3 || cols[2] == "" {
			continue
		}

		changes = append(changes, Change{
			Path:    cols[2],
			Added:   count(cols[0]),
			Deleted: count(cols[1]),
		})
	}

	return changes
}

// count reads one of numstat's two figures, and answers -1 for the "-" git
// prints where a file has no lines to count.
func count(field string) int {
	n, err := strconv.Atoi(strings.TrimSpace(field))
	if err != nil {
		return -1
	}

	return n
}

// HeadSHA is the commit a worktree stands at.
//
// The full hash and not the short one: it goes into the record as the way
// back, and an abbreviation that is unique today is one that stops being
// unique as the repository grows.
func (r Repo) HeadSHA(wtDir string) (string, error) {
	out, err := git(wtDir, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("read the head of %q: %w", wtDir, err)
	}

	return strings.TrimSpace(out), nil
}

// Backup writes a tag at a commit so that whatever happens next can be
// undone.
//
// A tag and not a branch: a branch is something git moves — a push, a
// checkout, a reset can all take it somewhere else — and a backup that moves
// is not a backup. The name is the caller's and carries the task and the
// hour, so two backups on one day do not overwrite each other.
func (r Repo) Backup(wtDir, name, at string) error {
	if _, err := git(wtDir, "tag", "-f", name, at); err != nil {
		return fmt.Errorf("tag %q at %q: %w", name, at, err)
	}

	return nil
}
