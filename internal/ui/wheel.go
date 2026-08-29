package ui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/layout"
)

// wheelRows is how far one notch of the wheel moves the list. Three is what
// a terminal's own scrollback moves and what every list in every other
// program moves, and a list that moved a different amount would be the one
// thing on screen the reader's hand has to learn.
const wheelRows = 3

// wheel scrolls whatever is under it, and only over the body.
//
// On the board it moves the cursor rather than the offset, because the
// offset is not the caller's to set: follow owns it, and it is what keeps
// the cursor on screen and the last page from scrolling past its end. In a
// pane it goes through the same scroll the arrow keys do, which is the one
// site the follow rule lives at — so a reader who wheels up a live log stops
// following it, exactly as a reader who presses ↑ does.
//
// Three notches of one row rather than one of three, so that a wheel and a
// held arrow key are the same gesture as far as everything downstream is
// concerned.
func (m Model) wheel(e tea.Mouse) Model {
	if m.frame.At(e.Y) != layout.RegionBody {
		return m
	}

	up := e.Button == tea.MouseWheelUp
	if !up && e.Button != tea.MouseWheelDown {
		// The wheel pushes sideways too, and nothing here scrolls that
		// way. The pane that can — a diff wider than the terminal — is
		// scrolled with ←→, and a sideways wheel is a gesture few mice
		// have and fewer readers expect.
		return m
	}

	if m.screen == screenDetail {
		// Anything the pane is made of: its own rows, the heads that fold
		// it, the bar down its edge. The wheel turns over all of them, and
		// a notch that did nothing because the pointer happened to rest on
		// a section head is a wheel that works in most of the window.
		switch m.hit(e.X, e.Y).Kind {
		case TargetPaneBody, TargetFold, TargetSeam, TargetPaneRow, TargetScrollBar:
		default:
			return m
		}

		k := firstKey(m.keys.Down)
		if up {
			k = firstKey(m.keys.Up)
		}

		for range wheelRows {
			m = m.scroll(k)
		}

		return m
	}

	if m.menu.open {
		// The menu's list moves its selection under the wheel, exactly as
		// the palette's does.
		d := wheelRows
		if up {
			d = -wheelRows
		}

		return m.menuPick(d)
	}

	if m.palette.open {
		// The wheel moves the selection, which is what scrolling a list
		// with one row chosen means; the list itself follows it through
		// ensureVisible rather than the reader following the list.
		d := wheelRows
		if up {
			d = -wheelRows
		}

		return m.pick(d)
	}

	if m.watchUp {
		// A run's output keeps its own tail on screen and offers nothing
		// to scroll back for yet, so the wheel does nothing here rather
		// than scrolling something that is not being shown.
		return m
	}

	if m.screen == screenSupervisor {
		// The thread scrolls under the wheel, and while a line is being
		// picked the wheel moves the pick instead — the same rule the
		// arrows follow on this screen, so the hand does not have to know
		// which of the two it is holding.
		d := wheelRows
		if up {
			d = -wheelRows
		}

		if m.supervisor.picking {
			m.supervisor.pick = min(max(m.supervisor.pick+d, 0), max(len(m.supervisor.lines)-1, 0))
			return m
		}

		return m.scrollThread(d)
	}

	if m.screen != screenList {
		return m
	}

	if up {
		return m.move(-wheelRows)
	}

	return m.move(wheelRows)
}

// firstKey is the keystroke a binding is reached by, which is the one a
// clicked hint sends and the one the wheel sends. A binding with no keys at
// all sends nothing rather than an empty keystroke every map would ignore
// anyway — the difference is that this one says so.
func firstKey(b key.Binding) keystroke {
	if keys := b.Keys(); len(keys) > 0 {
		return keystroke(keys[0])
	}

	return ""
}

// rowOf is which line of the body a target is on, by what it is rather than
// by where it was: the board is re-read twice a second, so the row a press
// landed on may have moved by the time the button comes up.
func (m Model) rowOf(t Target) (int, bool) {
	for i, r := range m.rows() {
		switch {
		case r.blank:
		case r.head && t.Kind == TargetBandHeader && r.band == t.Band:
			return i, true
		case !r.head && t.Kind == TargetTask && r.task.ID == t.ID:
			return i, true
		}
	}

	return 0, false
}

// same is whether two targets are the same thing, disregarding which field
// of it was pointed at.
func (t Target) same(o Target) bool {
	t.Column, o.Column = 0, 0
	return t == o
}

// paneBand is where the pane was drawn: the body row its first line landed
// on, and how many rows it got. It is the ruler a click on the scroll bar is
// measured against, and it is counted from the same rows detailRows draws.
func (m Model) paneBand() (top, rows int) {
	return m.paneBandFor(m.tab)
}

// paneBandFor is the same question asked of a tab that is not the one on
// show, which is what sizing every pane at once needs: the diff carries its
// file selector between the tab strip and its first line, so its pane is
// shorter than the others by however many rows that selector took.
//
// The model is a value, so pointing it at another tab to ask is a question
// and not a move.
func (m Model) paneBandFor(t tab) (top, rows int) {
	m.tab = t
	top = len(m.detailTop(m.frame.Body.H, m.frame.Body.W))

	return top, max(0, m.frame.Body.H-top-1)
}

// barShows says whether the pane has a bar down its edge to be pointed at,
// which is the same question scrollTrack answers by drawing one.
func (m Model) barShows() bool {
	_, rows := m.paneBand()

	return scrollTrack(rows, m.panes[m.tab].TotalLineCount(), 0) != nil
}

// scrollTo puts the pane where a click on its bar points: how far down the
// rail the pointer is, is how far into the text the top of the view goes.
//
// The thumb is not grabbed at the cell it was taken by. A pointer that has
// hold of the bar is read as a position and not as a distance, so a drag
// that leaves the rail and comes back is where it points rather than where
// it has been.
func (m Model) scrollTo(row int) Model {
	_, rows := m.paneBand()

	vp := m.panes[m.tab]

	total := vp.TotalLineCount()
	if rows <= 0 || total <= rows {
		return m
	}

	// The viewport holds both ends itself, so a pointer dragged off either
	// end of the rail asks for the end it went past.
	vp.SetYOffset(row * total / rows)
	m.panes[m.tab] = vp

	// The timeline follows the record while the view is at its end, and a
	// drag that leaves the end is the reader taking it back — the same rule
	// the arrow keys go through.
	if m.tab == tabTimeline {
		m.following = vp.AtBottom()
	}

	return m
}

// dragBar follows a bar that is being held. Only the row is read: a pointer
// that wanders off the rail with the button down is still dragging it, which
// is what every other scroll bar a reader has used does.
func (m Model) dragBar(e tea.Mouse) Model {
	line, ok := m.frame.BodyRow(e.Y)
	if !ok {
		return m
	}

	top, _ := m.paneBand()

	return m.scrollTo(line - top)
}
