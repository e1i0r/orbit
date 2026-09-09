package ui

// The repository as a tree, read for the map pane.
//
// The same reading the browser draws as a honeycomb, and the same builder:
// verb.Grow turns the paths and the change into one shape, so the two
// surfaces cannot disagree about what is in a repository. What differs is
// the drawing, because a terminal and a browser are good at different
// things — see internal/ui/panes/map.go for why this one is not hexagons.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/view"
)

// mapped is what the map pane holds. The three fields around the answer draw
// the distinction the impact pane's do: asked for, out, answered.
type mapped struct {
	tree   verb.Cell
	err    error
	known  bool
	asking bool
}

// treeMsg is the reading, come back.
type treeMsg struct {
	id   string
	tree verb.Cell
	err  error
}

// askTree reads it, unless it is already read or already out.
func (m Model) askTree() (Model, tea.Cmd) {
	if m.opts.Reader == nil || m.detail == "" || m.shape.known || m.shape.asking {
		return m, nil
	}

	t, held := m.task(m.detail)
	if !held || t.RepoPath == "" {
		return m, nil
	}

	m.shape.asking = true

	return m, treeOf(m.opts.Reader, t)
}

// treeOf reads what the checkout holds and what the task changed in it.
//
// The worktree is asked for through the port for the reason the diff asks
// for it: where a task's checkout lives is internal/store's answer, and this
// package may not name that.
func treeOf(r Reader, t view.Task) tea.Cmd {
	return func() tea.Msg {
		dir, err := r.Worktree(t.RepoPath, t.ID)
		if err != nil {
			return treeMsg{id: t.ID, err: err}
		}

		one := repo.Repo{Path: t.RepoPath, Name: t.Repo, Base: baseOf(t.RepoPath)}

		files, err := one.WorktreeFiles(dir)
		if err != nil {
			return treeMsg{id: t.ID, err: err}
		}

		// Beside the error rather than instead of the map: a change git
		// will not describe costs the marks, and a repository drawn with
		// nothing lit is still the answer to "what is in here".
		changes, _ := one.WorktreeChanges(dir) //nolint:errcheck // see above

		// The neighbours are read too, so that both surfaces are handed
		// the same tree. This pane does not draw them — a terminal has no
		// lattice to arrange — but a tree that differed between the two
		// would be the thing verb.Grow exists to stop.
		near, _ := one.Neighbours(dir) //nolint:errcheck // see above

		return treeMsg{id: t.ID, tree: verb.Grow(files, changes, near...)}
	}
}

// tookTree writes the reading in, if it is still the task on screen.
func (m Model) tookTree(msg treeMsg) Model {
	if msg.id != m.detail {
		return m
	}

	m.shape.tree, m.shape.err, m.shape.known, m.shape.asking = msg.tree, msg.err, true, false

	return m.syncPanes()
}

// forgetTree drops the reading when the view moves to another task.
func (m Model) forgetTree() Model {
	m.shape = mapped{}

	return m
}
