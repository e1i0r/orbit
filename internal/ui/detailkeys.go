package ui

// The task view's key map: the gestures that only exist one level down, and
// the one rule that keeps a live log honest.
//
// It is a file of its own because detail.go is the frame and this is the
// behaviour, and because both were over the line together. The split is the
// same one keypress.go and screen.go already make on the board.

import (
	"errors"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// openDetail opens the task view on one task, from the board.
func (m Model) openDetail(t view.Task) (Model, tea.Cmd) {
	m.screen, m.detail, m.tab = screenDetail, t.ID, tabLog
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
//
// Sideways is gated on the diff tab rather than ordered before Back, because
// ← means two different things depending on which pane is showing: on the
// diff it scrolls a line too wide for the pane, and on the other two tabs it
// is Back's own key, and pressing it there has to reach Back. Matching
// Sideways first and unconditionally — the previous shape of this switch —
// shadowed Back on every tab, not only the one that can scroll sideways, and
// turned ← on the log and evidence tabs into a silent no-op. esc is still
// the way out the key bar and the help overlay both print, so nothing on
// screen ever advertised ←; but it used to work, and this restores it.
func (m Model) detailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.tab == tabDiff && key.Matches(msg, m.keys.Sideways):
		return m.sideways(msg), nil
	case key.Matches(msg, m.keys.Back):
		m.screen = screenList
		return m, nil
	case key.Matches(msg, m.keys.NextTab):
		m.tab = (m.tab + 1) % tabCount
		return m, nil
	case key.Matches(msg, m.keys.PrevTab):
		m.tab = (m.tab + tabCount - 1) % tabCount
		return m, nil
	case key.Matches(msg, m.keys.Edit):
		return m.edit()
	case key.Matches(msg, m.keys.Open), key.Matches(msg, m.keys.Last):
		return m.newest(), nil
	case key.Matches(msg, m.keys.Help):
		// Answered here for the reason startKey answers it: the bar prints
		// [?] on every screen and never drops it, so every screen owes the
		// reader the same sentence back. Without this arm ? fell through to
		// scroll, which moves nothing and says nothing.
		return m.notBuilt(m.keys.Help), nil
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}
	return m.scroll(msg), nil
}

// scroll moves the pane, and is the one site the follow rule lives at.
//
// In the program this replaces the same rule was written at six places — one
// per key that could move the log — and they drifted: two of them let the
// tail go and never took it back, so a reader who pressed PgUp once stopped
// seeing new output for the rest of the run and had no way to know. Here
// every key that can move the pane comes through this function, and the rule
// is read off the offset afterwards rather than written into each key.
func (m Model) scroll(msg tea.KeyPressMsg) Model {
	vp := m.panes[m.tab]
	was := vp.YOffset()
	switch {
	case key.Matches(msg, m.keys.Up):
		vp.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		vp.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		vp.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		vp.PageDown()
	case key.Matches(msg, m.keys.First):
		vp.GotoTop()
	default:
		return m
	}
	m.panes[m.tab] = vp
	if m.tab != tabLog {
		return m
	}
	if vp.YOffset() < was {
		m.following = false
	} else if vp.AtBottom() {
		m.following = true
	}
	return m
}

// sideways scrolls the pane along a line too wide for it, which only the
// diff ever is.
func (m Model) sideways(msg tea.KeyPressMsg) Model {
	vp := m.panes[m.tab]
	if msg.Code == tea.KeyLeft {
		vp.ScrollLeft(sidewaysStep)
	} else {
		vp.ScrollRight(sidewaysStep)
	}
	m.panes[m.tab] = vp
	return m
}

// sidewaysStep is how far one press moves along a line. Eight cells is one
// level of indentation in most of what a diff contains, and one cell at a
// time across a four-thousand-cell line is not scrolling.
const sidewaysStep = 8

// newest jumps to the end of the pane and arms the tail again. It is what ⏎
// does here, which is the same thing it does on the board: go to the thing
// that is happening now.
func (m Model) newest() Model {
	vp := m.panes[m.tab]
	vp.GotoBottom()
	m.panes[m.tab] = vp
	if m.tab == tabLog {
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
