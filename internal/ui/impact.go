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
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// weighed is what the impact pane holds.
//
// The three fields around each answer draw the distinction the diff's do:
// asked for, out, answered, refused. Nothing found and not looked yet are
// different facts, and a pane that folded them into one would tell a reader
// a repository has no coupling when what happened is that git timed out.
type weighed struct {
	reach       repo.Impact
	reachErr    error
	reachKnown  bool
	reachAsking bool
	checks      []repo.Divergence
	checksErr   error
	checksKnown bool
	running     bool
	since       time.Time
	// reread is whether the reading has already been taken a second time
	// because the diff disagreed with it. It is what keeps that second
	// reading from becoming an every-two-seconds one: the diff is polled on
	// a clock, and a disagreement neither side can settle would otherwise
	// send five hundred commits of git log after every poll — which is what
	// it did, and the pane spent its life loading rather than being read.
	reread bool
}

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
	if m.opts.Reader == nil || m.detail == "" || m.weigh.reachKnown || m.weigh.reachAsking {
		return m, nil
	}

	t, held := m.task(m.detail)
	if !held || t.RepoPath == "" {
		return m, nil
	}

	m.weigh.reachAsking = true

	return m, impactOf(m.opts.Reader, t)
}

// impactOf reads the history behind one task's worktree.
//
// The base branch travels with the repository, as it does for the comparison
// and for the diff. Without it the reading asks git what is uncommitted, and
// a task that committed its work — which is every task that finished — has
// nothing uncommitted: the pane said "this change touched no files" beside a
// diff of nineteen, and the disagreement sent it back to git on every poll.
func impactOf(r Reader, t view.Task) tea.Cmd {
	return func() tea.Msg {
		dir, err := r.Worktree(t.RepoPath, t.ID)
		if err != nil {
			return impactMsg{id: t.ID, err: err}
		}

		got, err := repo.Repo{Path: t.RepoPath, Name: t.Repo, Base: baseOf(t.RepoPath)}.Impact(dir)

		return impactMsg{id: t.ID, impact: got, err: err}
	}
}

// tookImpact writes the reading in, if it is still the task on screen.
func (m Model) tookImpact(msg impactMsg) Model {
	if msg.id != m.detail {
		return m
	}

	m.weigh.reach, m.weigh.reachErr, m.weigh.reachKnown, m.weigh.reachAsking = msg.impact, msg.err, true, false

	return m.syncPanes()
}

// forgetImpact drops the reading when the view moves to another task.
func (m Model) forgetImpact() Model {
	m.weigh.reach, m.weigh.reachErr = repo.Impact{}, nil
	m.weigh.reachKnown, m.weigh.reachAsking = false, false
	m.weigh.reread = false

	return m
}

// staleImpact is whether the reading and the diff disagree about whether
// this task has changed anything.
//
// It is the one cheap signal that the history was read too early: reading it
// again costs five hundred commits of git log, so it is not done on a clock
// — the diff is polled every couple of seconds and that would be a subprocess
// every couple of seconds for the life of the view.
func (m Model) staleImpact() bool {
	if !m.weigh.reachKnown || m.weigh.reachErr != nil || m.weigh.reread {
		return false
	}

	return (len(m.weigh.reach.Changed) == 0) != (strings.TrimSpace(m.diff) == "")
}

// impactWarnings is how many things the reading found, for the mark on the
// tab. Nothing found draws no mark: a zero on a tab strip is a number nobody
// needs to read.
func (m Model) impactWarnings() int {
	if !m.weigh.reachKnown || m.weigh.reachErr != nil {
		return 0
	}

	return len(m.weigh.reach.Coupled)
}
