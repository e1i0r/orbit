package ui

// The menu inside a task: the panes it can be shown in, and what can be
// done to it, under a heading each.
//
// One list with two kinds of thing in it needs the reader told which is
// which. A menu that opened on a task and offered eleven panes and nothing
// else said, by omission, that looking is all there is to do in here — the
// verbs were on the board's menu, one screen back, which is not where
// somebody reading a run reaches for them.

// detailMenuEntries is that menu: the panes, then the verbs, each behind a
// line naming what the block is.
//
// The panes come first because they are what the keystroke that opens this
// menu has always led to, and a reader who learned the list in that order
// should not have to find it again underneath a block that grew above it.
func (m Model) detailMenuEntries() []menuEntry {
	p := m.opts.Words

	out := []menuEntry{menuHead(p.T("menu.head_panes", "the panes of this task"))}
	out = append(out, m.tabMenuEntries()...)

	verbs := m.taskVerbEntries(m.menu.taskID)
	if len(verbs) == 0 {
		return out
	}

	out = append(out, menuGap(), menuHead(p.T("menu.head_verbs", "what can be done to it")))

	return append(out, verbs...)
}

// menuHead is a line that names a block rather than offering anything. It is
// skipped by the cursor and answers nothing to a click, because a heading a
// reader can land on is a row that looks chosen and does nothing.
func menuHead(title string) menuEntry { return menuEntry{head: true, title: title} }

// menuGap is the blank between two blocks. It is an entry rather than a line
// the drawing adds, because the menu is hit-tested by counting rows: a line
// drawn that no entry accounts for puts every click below it on the wrong
// verb.
func menuGap() menuEntry { return menuEntry{head: true} }

// taskVerbEntries is what can be done to one task, refusals included and
// each with its reason.
//
// It is the same list on the board's menu and inside a task, from the same
// call, because they are the same question asked from two places — and a
// verb offered in one and missing from the other would be read as a verb
// that does not apply here.
func (m Model) taskVerbEntries(id string) []menuEntry {
	t, ok := m.task(id)
	if !ok {
		return nil
	}

	p := m.opts.Words
	all := m.keys.Affordances(t, m.conditions(t))

	out := make([]menuEntry, 0, len(all))

	for i := range all {
		a := &all[i]

		e := menuEntry{
			glyph: a.Key.Help().Key,
			title: a.Key.Help().Desc,
			aff:   a,
		}
		if !a.OK {
			e.dim = true
			e.reason = a.Why(p)
		}

		out = append(out, e)
	}

	// And what only a command or a dialog can do to it. They are in the same
	// block rather than under a heading of their own: a reader asking what
	// can be done to a task is not asking which of the answers is a key.
	out = append(out, m.startEntry())

	return append(out, m.taskCommandEntries(id)...)
}

// menuChoice is the entry the cursor may sit on nearest to from, walking in
// direction d and stopping at whichever end it reaches. Heads and gaps are
// passed over, and a list that is nothing but heads has no choice in it,
// which is what -1 says.
func menuChoice(es []menuEntry, from, d int) int {
	for i := from; i >= 0 && i < len(es); i += d {
		if !es[i].head {
			return i
		}
	}

	// The end of the list in that direction was a heading. The cursor stays
	// where it can be rather than parking on a line that does nothing.
	for i := from; i >= 0 && i < len(es); i -= d {
		if !es[i].head {
			return i
		}
	}

	return -1
}

// menuHeadRow draws a heading: no gutter, no glyph, the accent the sections
// of the knobs screen are named in.
func menuHeadRow(e menuEntry, w int) string {
	if e.title == "" {
		return ""
	}

	return fit("  "+Paint(Accent).Render(e.title), w)
}
