package ui

// Running the flow's own checks on both sides of the change.
//
// It is asked for and not taken: a test suite run twice is minutes and money
// on somebody's machine, and it may touch a database or a network — the
// commands belong to the repository, and Orbit does not decide on its own to
// run them again. So the pane says what it would run, and a key runs it.

import (
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// comparedMsg is the comparison, come back.
type comparedMsg struct {
	id  string
	got []repo.Divergence
	err error
}

// checksOf is what this task's flow already trusts: every gate of every
// phase, and the checks a loop stops on.
//
// The flow's own and nothing invented. A command guessed from the shape of
// the repository — a go.mod here, a package.json there — is Orbit deciding
// what this project's tests are, and being wrong about it costs ten minutes
// and prints something nobody asked for.
func (m Model) checksOf(t view.Task) []repo.Check {
	if m.opts.Flows == nil || t.Flow == "" {
		return nil
	}

	fl, err := flow.Resolve(m.opts.Flows, t.Flow)
	if err != nil {
		return nil
	}

	var out []repo.Check

	for _, p := range fl.Phases {
		out = append(out, checksIn(p)...)
	}

	return out
}

// checksIn is one phase's checks: its gates, and the ones a loop inside it
// stops on.
func checksIn(p flow.Phase) []repo.Check {
	var out []repo.Check

	for _, g := range p.Gates {
		out = append(out, repo.Check{Name: g.Name, Command: g.Command})
	}

	if p.Loop == nil {
		return out
	}

	for _, g := range p.Loop.Until {
		out = append(out, repo.Check{Name: g.Name, Command: g.Command})
	}

	for _, inner := range p.Loop.Phases {
		out = append(out, checksIn(inner)...)
	}

	return out
}

// compareSides runs them, on the base and on the work.
func (m Model) compareSides() (Model, tea.Cmd) {
	p := m.opts.Words
	if m.weigh.running || m.opts.Reader == nil {
		return m, nil
	}

	t, held := m.task(m.detail)
	if !held || t.RepoPath == "" {
		return m, nil
	}

	checks := m.checksOf(t)
	if len(checks) == 0 {
		return m.say(p.T("compare.no_checks",
			"this task's flow has no checks, so there is nothing to run on either side")), nil
	}

	m.weigh.running, m.weigh.since = true, m.now
	m = m.say(p.T("compare.running", "running {n} checks on both sides",
		about("n", strconv.Itoa(len(checks)))))

	next, frame := m.nextFrame()
	reader := m.opts.Reader

	return next, tea.Batch(frame, func() tea.Msg {
		dir, err := reader.Worktree(t.RepoPath, t.ID)
		if err != nil {
			return comparedMsg{id: t.ID, err: err}
		}

		got, err := repo.Repo{Path: t.RepoPath, Name: t.Repo, Base: baseOf(t.RepoPath)}.Compare(dir, checks)

		return comparedMsg{id: t.ID, got: got, err: err}
	})
}

// tookComparison writes the answer in, if it is still the task on screen.
func (m Model) tookComparison(msg comparedMsg) Model {
	m.weigh.running = false

	if msg.id != m.detail {
		return m
	}

	m.weigh.checks, m.weigh.checksErr, m.weigh.checksKnown = msg.got, msg.err, true

	if msg.err != nil {
		return m.syncPanes().say(m.opts.Words.T("compare.failed", "the checks could not be run: {err}",
			about("err", m.errSaid(msg.err))))
	}

	return m.syncPanes().say(m.opts.Words.T("compare.done", "{n} of {total} checks answered differently",
		about("n", strconv.Itoa(len(diverged(msg.got)))), about("total", strconv.Itoa(len(msg.got)))))
}

// forgetComparison drops it when the view moves to another task.
func (m Model) forgetComparison() Model {
	m.weigh.checks, m.weigh.checksErr = nil, nil
	m.weigh.checksKnown, m.weigh.running = false, false

	return m
}

// diverged is the checks that answered differently on the two sides.
func diverged(all []repo.Divergence) []repo.Divergence {
	var out []repo.Divergence

	for _, d := range all {
		if !d.Same() {
			out = append(out, d)
		}
	}

	return out
}

// comparedFor is how long the run has been out, for the line that says so.
func (m Model) comparedFor() time.Duration {
	if m.weigh.since.IsZero() {
		return 0
	}

	return m.now.Sub(m.weigh.since).Round(time.Second)
}

// compareSidesCmd is the key, as the task view's map takes it.
func (m Model) compareSidesCmd() (tea.Model, tea.Cmd) {
	next, cmd := m.compareSides()

	return next, cmd
}
