package ui

// The task view's key map: the gestures that only exist one level down, and
// the one rule that keeps a live log honest.
//
// It is a file of its own because detail.go is the frame and this is the
// behaviour, and because both were over the line together. The split is the
// same one keypress.go and screen.go already make on the board.

import (
	"errors"
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// openDetail opens the task view on one task, from the board.
func (m Model) openDetail(t view.Task) (Model, tea.Cmd) {
	m.screen, m.detail, m.tab = screenDetail, t.ID, tabOverview
	m.entries, m.logErr, m.diff, m.following = nil, nil, "", true
	m.diffErr, m.diffKnown, m.diffNoBase = nil, false, false
	// The base is one of the things an open forgets: it belongs to the
	// repository this task is in, and asking for it again is the one thing
	// this window does per open rather than per tick.
	m.diffBase, m.diffAsking = baseRef{}, true
	for i := range m.panes {
		m.panes[i] = viewport.New()
	}
	return m.syncPanes(), tea.Batch(logOf(m.opts.Reader, t), diffOf(m.opts.Reader, t, m.diffBase))
}

// detailKey is the task view's map.
func (m Model) detailKey(k fmt.Stringer) (tea.Model, tea.Cmd) {
	if targetTab, ok := keyToPane(k.String()); ok {
		return m.showTab(targetTab), nil
	}
	switch {
	case m.tab == tabDiff && key.Matches(k, m.keys.Sideways):
		return m.sideways(k), nil
	case key.Matches(k, m.keys.Back):
		m.screen = screenList
		return m, nil
	case key.Matches(k, m.keys.NextTab):
		return m.showTab((m.tab + 1) % tabCount), nil
	case key.Matches(k, m.keys.PrevTab):
		return m.showTab((m.tab + tabCount - 1) % tabCount), nil
	case key.Matches(k, m.keys.Edit):
		return m.edit()
	case key.Matches(k, m.keys.Menu), k.String() == "m", k.String() == "M":
		return m.openMenuForContext(), nil
	case key.Matches(k, m.keys.Ask):
		return m.openNote(), nil
	case key.Matches(k, m.keys.CLI):
		return m.launchInteractiveCLI()
	case key.Matches(k, m.keys.Open), key.Matches(k, m.keys.Last):
		return m.newest(), nil
	case key.Matches(k, m.keys.Help):
		return m.openHelp(), nil
	case key.Matches(k, m.keys.Quit):
		return m, tea.Quit
	}
	return m.scroll(k), nil
}

// showTab puts one pane on top.
func (m Model) showTab(t tab) Model {
	m.tab = t
	return m
}

// scroll moves the pane, and is the one site the follow rule lives at.
func (m Model) scroll(k fmt.Stringer) Model {
	vp := m.panes[m.tab]
	was := vp.YOffset()
	switch {
	case key.Matches(k, m.keys.Up):
		vp.ScrollUp(1)
	case key.Matches(k, m.keys.Down):
		vp.ScrollDown(1)
	case key.Matches(k, m.keys.PageUp):
		vp.ScrollUp(m.frame.Body.H)
	case key.Matches(k, m.keys.PageDown):
		vp.ScrollDown(m.frame.Body.H)
	case key.Matches(k, m.keys.First):
		vp.GotoTop()
	case key.Matches(k, m.keys.Last):
		vp.GotoBottom()
	default:
		return m
	}
	m.panes[m.tab] = vp
	if m.tab == tabTimeline {
		if vp.AtBottom() {
			m.following = true
		} else if vp.YOffset() < was {
			m.following = false
		}
	}
	return m
}

// sideways scrolls the pane along a line too wide for it, which only the
// diff ever is.
func (m Model) sideways(k fmt.Stringer) Model {
	vp := m.panes[m.tab]
	if k.String() == "left" {
		vp.ScrollLeft(sidewaysStep)
	} else {
		vp.ScrollRight(sidewaysStep)
	}
	m.panes[m.tab] = vp
	return m
}

// sidewaysStep is how far one press moves along a line.
const sidewaysStep = 8

// newest jumps to the end of the pane and arms the tail again.
func (m Model) newest() Model {
	vp := m.panes[m.tab]
	vp.GotoBottom()
	m.panes[m.tab] = vp
	if m.tab == tabTimeline {
		m.following = true
	}
	return m
}

// logOf reads one task's record, off the event loop.
func logOf(r Reader, t view.Task) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return logMsg{ID: t.ID, Err: errors.New("this window was opened without a way to read the record")}
		}
		entries, err := r.Log(t.RepoPath, t.ID)
		if err != nil {
			return logMsg{ID: t.ID, Err: err}
		}
		return logMsg{ID: t.ID, Entries: entries}
	}
}
