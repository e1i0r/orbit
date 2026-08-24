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
		if m.hit(e.X, e.Y).Kind != TargetPaneBody {
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
