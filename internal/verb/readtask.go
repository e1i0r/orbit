package verb

// The readings that are about one task: what it is set up to do, what it
// changed, and how far that change reaches.
//
// Each answers twice, the way every reading here does: Saw is the thing
// itself, for a surface that draws structures, and Said is the same answer
// written out for a terminal.

import (
	"fmt"
	"os"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/task"
)

// shaped is the flow a task walks, and what each of its phases is set up to
// do.
//
// How far a run got through it is not here: that is the window's flow tab
// and the browser's flow screen, both of which draw gates, loops and cost
// beside each phase. This is the shape, which is what a reader at a terminal
// is asking for when they ask what a task is going to do.
func shaped(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	// The task's own flow, then the one Orbit ships — the same reading
	// `orbit run` makes, and not the settings default, which is what the
	// next task written gets rather than what this one walks.
	chosen := t.Flow
	if chosen == "" {
		chosen = flow.Default
	}

	f, err := flow.Resolve(w.Store(), chosen)
	if err != nil {
		return Out{}, err
	}

	var b strings.Builder

	fmt.Fprintf(&b, "%s: %s\n", t.ID, f.Name)

	for i, p := range f.Phases {
		fmt.Fprintf(&b, "%2d  %-16s %s\n", i+1, p.Name, p.Engine)
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: f}, nil
}

// changed is what a task changed in its checkout, as git wrote it.
//
// Handed over as it came. Turning a stream into files, hunks and lines is
// the reader's job, and a browser has a whole language for that where a
// terminal has git's own.
func diffed(w World, in In) (Out, error) {
	t, dir, err := checkout(w, in)
	if err != nil {
		return Out{}, err
	}

	if dir == "" {
		return Out{Said: t.ID + " has no checkout to read"}, nil
	}

	one, err := repo.Open(t.Repo.Path)
	if err != nil {
		return Out{}, err
	}

	text, err := one.WorktreeDiff(dir, repo.DiffOptions{IgnoreWhitespace: in.Yes("space")})
	if err != nil {
		return Out{}, err
	}

	if strings.TrimSpace(text) == "" {
		return Out{Said: t.ID + " has changed nothing yet", Saw: ""}, nil
	}

	return Out{Said: text, Saw: text}, nil
}

// reaches is what a change touches beyond the files it changed.
//
// Three claims of three different weights, kept apart in the answer: what
// the history says usually comes along, what the tests among those files are
// named after, and the engine's own account of what its change asks and
// promises. Nothing here was run.
func reaches(w World, in In) (Out, error) {
	t, dir, err := checkout(w, in)
	if err != nil {
		return Out{}, err
	}

	if dir == "" {
		return Out{Said: t.ID + " has no checkout to read"}, nil
	}

	one, err := repo.Open(t.Repo.Path)
	if err != nil {
		return Out{}, err
	}

	got, err := one.Impact(dir)
	if err != nil {
		return Out{}, err
	}

	return Out{Said: reads(got), Saw: got}, nil
}

// reads is one impact as it prints on a terminal.
func reads(got repo.Impact) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%d files changed, read against %d commits\n", len(got.Changed), got.Commits)

	for _, c := range got.Coupled {
		fmt.Fprintf(&b, "  %s usually changes with %s (%d of %d)\n", c.With, c.File, c.Times, c.Of)
	}

	for _, c := range got.Contracts {
		fmt.Fprintf(&b, "  %s holds: %s\n", c.File, c.Says)
	}

	return strings.TrimRight(b.String(), "\n")
}

// checkout is the task and the directory its work is in, and an empty
// directory for a task that has none yet.
//
// A missing checkout is not a failure. A task written and never run has no
// worktree, and "there is nothing to read" is the true answer to what it
// changed.
func checkout(w World, in In) (task.Task, string, error) {
	t, err := found(w, in)
	if err != nil {
		return task.Task{}, "", err
	}

	if t.Repo.Path == "" {
		return t, "", nil
	}

	dir, err := w.Store().WorktreeDir(t.Repo.Path, t.ID)
	if err != nil {
		return t, "", fmt.Errorf("locate the worktree of task %s: %w", t.ID, err)
	}

	// A path the disk does not answer for is a task that has not run yet,
	// not a failure: the worktree is made by the first phase, and "there is
	// nothing to read" is the true answer until then.
	if _, err := os.Stat(dir); err != nil {
		return t, "", nil //nolint:nilerr // see above
	}

	return t, dir, nil
}
