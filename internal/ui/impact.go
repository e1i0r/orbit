package ui

// The impact pane: what this change reaches beyond the files it touched.
//
// A diff answers "what did it write". This answers the question nobody can
// read off a diff — what usually moves with these files and did not this
// time — and it answers it from the repository's own history rather than
// from anything Orbit believes about the code.
//
// It is asked for once per task, in the background, and only for a task with
// changes: reading five hundred commits is a second on a large repository
// and this screen redraws ten times a second.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// impactMsg is the reading, come back.
type impactMsg struct {
	id     string
	impact repo.Impact
	err    error
}

// askImpact reads it, unless it is already read or already out.
//
// The worktree is asked for through the port for the reason the diff asks
// for it: where a task's checkout lives is internal/store's answer, and this
// package may not name that. Reading the repository instead would weigh
// whatever the reader happens to have uncommitted in their own checkout.
func (m Model) askImpact() (Model, tea.Cmd) {
	if m.opts.Reader == nil || m.detail == "" || m.impactKnown || m.impactAsking {
		return m, nil
	}

	t, held := m.task(m.detail)
	if !held || t.RepoPath == "" {
		return m, nil
	}

	m.impactAsking = true

	return m, impactOf(m.opts.Reader, t)
}

// impactOf reads the history behind one task's worktree.
func impactOf(r Reader, t view.Task) tea.Cmd {
	return func() tea.Msg {
		dir, err := r.Worktree(t.RepoPath, t.ID)
		if err != nil {
			return impactMsg{id: t.ID, err: err}
		}

		got, err := repo.Repo{Path: t.RepoPath, Name: t.Repo}.Impact(dir)

		return impactMsg{id: t.ID, impact: got, err: err}
	}
}

// tookImpact writes the reading in, if it is still the task on screen.
func (m Model) tookImpact(msg impactMsg) Model {
	if msg.id != m.detail {
		return m
	}

	m.impact, m.impactErr, m.impactKnown, m.impactAsking = msg.impact, msg.err, true, false

	return m.syncPanes()
}

// forgetImpact drops the reading when the view moves to another task.
func (m Model) forgetImpact() Model {
	m.impact, m.impactErr = repo.Impact{}, nil
	m.impactKnown, m.impactAsking = false, false

	return m
}

// impactWarnings is how many things the reading found, for the mark on the
// tab. Nothing found draws no mark: a zero on a tab strip is a number nobody
// needs to read.
func (m Model) impactWarnings() int {
	if !m.impactKnown || m.impactErr != nil {
		return 0
	}

	return len(m.impact.Coupled)
}
