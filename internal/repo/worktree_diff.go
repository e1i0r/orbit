package repo

// What a task's worktree has changed, counted rather than shown.

import (
	"fmt"
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

	args := append([]string{"diff", "--numstat"}, r.against(wtDir)...)

	out, err := git(wtDir, args...)
	if err != nil {
		// Without a base there is nothing to compare against and the
		// working tree is the whole answer — a worktree cut before its base
		// existed, or one whose base has been deleted since, still has
		// changes worth counting.
		out, err = git(wtDir, "diff", "--numstat")
		if err != nil {
			return nil, fmt.Errorf("count what %q changed: %w", wtDir, err)
		}
	}

	return numstat(out), nil
}

// WorktreeAddedLines is every line the task added to one file, without the
// leading plus.
//
// Added and not changed: what a dependency gate asks is what appeared, and a
// line that was removed cannot have brought a library with it. -U0 so that
// the context lines around a change — which are somebody else's
// dependencies, already there — are not read as new ones.
func (r Repo) WorktreeAddedLines(wtDir, path string) ([]string, error) {
	args := append([]string{"diff", "-U0"}, r.against(wtDir)...)
	args = append(args, "--", path)

	out, err := git(wtDir, args...)
	if err != nil {
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
	if r.Base == "" {
		return nil
	}

	if tracking := r.Remote + "/" + r.Base; r.Remote != "" && r.resolves(wtDir, tracking) {
		return []string{"--merge-base", tracking}
	}

	return []string{"--merge-base", r.Base}
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
