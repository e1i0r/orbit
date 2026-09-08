package ui

// Where the window and the form a task is written into meet.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/compose"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// composeEnv is the world the form was written against.
func (m Model) composeEnv() compose.Env {
	listed := m.collectRepos()

	places := make([]compose.Place, 0, len(listed))
	for _, r := range listed {
		places = append(places, compose.Place{Name: r.Name, Path: r.Path})
	}

	return compose.Env{
		Words:     m.opts.Words,
		Keys:      m.keys,
		Frame:     m.frame,
		Now:       m.now,
		Flows:     m.opts.Flows,
		Places:    places,
		Autopilot: m.autopilotOn(),
		ValidID:   m.opts.ValidID,
	}
}

// tookCompose does what the form asked the window for.
func (m Model) tookCompose(next compose.State, out compose.Out) (tea.Model, tea.Cmd) {
	m.compose = next

	if out.Leave {
		m.screen = screenList
	}

	if out.Said != "" {
		m = m.say(out.Said)
	}

	if out.Autopilot {
		return m.autopilot()
	}

	if out.Flow != "" {
		if out.Flow == compose.New {
			return m.openFlows(), nil
		}

		return m.openFlowPreview(out.Flow), nil
	}

	cmd := out.Cmd

	if out.Waiting {
		started, frame := m.nextFrame()
		m, cmd = started, tea.Batch(cmd, frame)
	}

	if out.Write == nil {
		return m, cmd
	}

	return m.writeTask(*out.Write)
}

// writeTask runs the command that writes a task down, and waits for the row
// to arrive: the board polls twice a second, so the new task is selected the
// moment it shows up.
func (m Model) writeTask(t compose.Task) (tea.Model, tea.Cmd) {
	args := []string{"-id", t.ID}
	if t.Repo != "" {
		args = append(args, "-repo", t.Repo)
	}

	if t.Flow != "" {
		args = append(args, "-flow", t.Flow)
	}

	if t.Start {
		args = append(args, "-start")
	}

	args = append(args, "--", t.Text)

	m.pendingID, m.pendTries = t.ID, 0

	return m.runWatched(Command{Name: "new"}, args)
}

// openCompose brings the form up, on the repository the cursor was over.
func (m Model) openCompose() Model {
	under := ""
	if r, ok := m.selected(); ok && !r.head {
		under = r.task.Repo
	}

	m.compose = compose.Open(under, m.composeEnv())
	m.screen = screenCompose

	return m
}

// openComposeFor brings the form up on a directory an interactive session
// was held in, with the caret already in the box the task is written in —
// where it was held is the answer to everything else.
//
// A session that remembered no directory is not a reason to forget the
// cursor: the form falls back to what it would have picked on its own.
func (m Model) openComposeFor(repo string) Model {
	if repo == "" {
		return m.openCompose()
	}

	m.compose = compose.OpenIn(repo, m.composeEnv())
	m.screen = screenCompose

	return m
}

// composeKey hands one keystroke to the form.
func (m Model) composeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, out := m.compose.Key(msg, m.composeEnv())

	return m.tookCompose(next, out)
}

// handleComposeClick hands one click to the form.
func (m Model) handleComposeClick(t point.Target) (tea.Model, tea.Cmd) {
	next, out := m.compose.Click(t, m.composeEnv())

	return m.tookCompose(next, out)
}

// tookIssue takes the tracker's answer back into the form.
func (m Model) tookIssue(msg compose.ReadMsg) (tea.Model, tea.Cmd) {
	next, out := m.compose.Took(msg, m.composeEnv())

	return m.tookCompose(next, out)
}

// composeAim puts the caret where the pointer is.
func (m Model) composeAim(t point.Target) Model {
	m.compose = m.compose.Aim(t, m.composeEnv())

	return m
}

// dragCaret is the pointer moving with the button still down: the caret
// follows it while the anchor stays on the cell the button went down on,
// which is a selection being dragged out of the text.
//
// A pointer that has wandered out of the field it started in is ignored
// rather than clamped to its edge. The drag is over that field's text, and a
// cell somewhere else on the form is not a place in it.
func (m Model) dragCaret(e tea.Mouse) Model {
	t := m.hit(e.X, e.Y)
	if t.Kind != point.ComposeCaret || t.Pane != m.held.target.Pane {
		return m
	}

	m.compose = m.compose.Drag(t, m.composeEnv())

	return m
}

// composeRows is the form drawn.
func (m Model) composeRows(h, w int) []string {
	return m.compose.View(h, w, m.composeEnv())
}

// hitCompose is what the form has at that cell.
func (m Model) hitCompose(x, y int) point.Target {
	return m.compose.Hit(x, y, m.composeEnv())
}

// selectPending waits for a newly written task to appear on the board.
func (m Model) selectPending() Model {
	if m.pendingID == "" {
		return m
	}

	for i, r := range m.rows() {
		if !r.head && !r.blank && r.task.ID == m.pendingID {
			m.pendingID, m.pendTries = "", 0

			return m.moveTo(i)
		}
	}

	m.pendTries++
	if m.pendTries > 2 {
		m.pendingID, m.pendTries = "", 0
	}

	return m
}
