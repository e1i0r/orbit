package ui

// The menu follows its cursor down a list longer than the window.

// menuView is how many entries fit under the title. The title and the blank
// under it do not scroll: a list that carried its own name off the top would
// leave the reader looking at rows with nothing saying what they are of.
func menuView(h int) int { return max(1, h-menuTitleRows) }

// menuOffset is the first entry drawn, clamped to a list that may have got
// shorter since it was set — the entries are recomputed on every frame, and
// a run that leaves the board takes its verbs with it.
func (m Model) menuOffset(entries, view int) int {
	return min(max(m.menu.offset, 0), max(0, entries-view))
}

// keepMenuEntrySeen moves the list as little as it can to keep the selection
// on screen. It is the palette's rule and the knobs': the reader is moving a
// cursor, and the scrolling is the screen's business rather than theirs.
func (m Model) keepMenuEntrySeen() Model {
	es := m.menuEntries()

	view := menuView(m.frame.Body.H)
	off := m.menuOffset(len(es), view)

	switch {
	case m.menu.sel < off:
		// A heading is shown with the entry under it, for the reason the
		// heading exists: "pause" at the top of the screen does not say
		// whether it is a pane or a verb.
		m.menu.offset = m.menu.sel
		if m.menu.sel > 0 && es[m.menu.sel-1].head {
			m.menu.offset = m.menu.sel - 1
		}
	case m.menu.sel >= off+view:
		m.menu.offset = m.menu.sel - view + 1
	default:
		m.menu.offset = off
	}

	if m.menu.offset < 0 {
		m.menu.offset = 0
	}

	return m
}
