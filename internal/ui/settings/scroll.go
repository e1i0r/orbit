package settings

// The window: which part of the table is on the screen, and keeping the row
// the cursor is on inside it.
//
// The table grew past the screen without anybody deciding it should. Nine
// dials at three lines each is twenty-seven rows against a body of about
// twenty, so the last few were drawn nowhere at all — reachable by a cursor
// that walked off the bottom and left no mark of having gone.

// The three parts of the screen, in lines: the head is the title and the
// blank rows around it, the foot is the line saying how to leave, and a row
// is its name, the sentence under it, and the blank that separates it from
// the next.
const (
	headLines = 4
	rowLines  = 3
	footLines = 1
)

// room is how many lines of the table the screen has left over for it.
func room(h int) int { return max(1, h-headLines-footLines) }

// window is the first line of the table on show, held inside a table that
// may have got shorter since it was set: the model and effort dials are the
// engine's, so choosing another engine can shorten no row and still leave
// the view parked past the end — which draws as blank rows with nothing on
// the screen saying why.
func window(off, lines, view int) int {
	return min(max(off, 0), max(0, lines-view))
}

// Off is the first line of the table on show.
//
// It is a door because a click is measured from it: the row under the
// pointer is the line it landed on plus however far the table has been
// scrolled, and a window that did not add it would turn the dial of
// whichever row used to be there.
func (s State) Off(e Env) int {
	if e.Frame.Body.H <= 0 {
		return 0
	}

	return window(s.off, rowLines*len(s.Rows(e)), room(e.Frame.Body.H))
}

// Scroll is the wheel. One notch is one setting and not three lines: a
// setting is three lines tall, so counted in lines the wheel crawls through
// a single dial.
//
// It moves the cursor and lets the table follow, which is what the wheel
// does on the engines screen and in the palette. A wheel that moved the view
// on its own would leave the cursor off the screen, and the next arrow key
// would snap the whole table back to it.
func (s State) Scroll(d int, e Env) State {
	rows := s.Rows(e)
	if len(rows) == 0 {
		return s
	}

	s.sel = min(max(s.sel+d, 0), len(rows)-1)

	return s.keepSeen(e)
}

// keepSeen moves the table as little as it can to keep the row the cursor is
// on in sight. The reader is moving a cursor; the scrolling is this screen's
// business rather than theirs.
func (s State) keepSeen(e Env) State {
	// A window that has not been sized yet holds nothing, and the height
	// this would scroll against is not the height View will be given. The
	// table starts at the top until there is a body to keep it inside.
	rows := s.Rows(e)
	if len(rows) == 0 || s.sel < 0 || s.sel >= len(rows) || e.Frame.Body.H <= 0 {
		s.off = 0

		return s
	}

	lines, view := rowLines*len(rows), room(e.Frame.Body.H)
	off := window(s.off, lines, view)

	// The blank under a row is not part of what has to be seen. A dial with
	// its name and its sentence both on the screen has been read, and
	// scrolling one line further to show the gap beneath it would move the
	// table for nothing.
	from, last := s.sel*rowLines, s.sel*rowLines+1

	switch {
	case from < off:
		off = from
	case last >= off+view:
		off = last - view + 1
	}

	s.off = window(atTheTop(off, view), lines, view)

	return s
}

// atTheTop puts the first line on show at the start of a row rather than in
// the middle of one.
//
// A view that lands mid-row draws that row's blank separator under the
// title, which reads as a gap somebody left there and spends a line of the
// table on nothing. Rounding up is always safe: the cursor's row starts on a
// multiple of three, so a view that had room for it below the old top still
// has room for it below this one.
//
// Not on a screen with less room than a row, where rounding up would push
// the row the cursor is on off the top of the view it was brought into.
func atTheTop(off, view int) int {
	if view < rowLines {
		return off
	}

	return (off + rowLines - 1) / rowLines * rowLines
}

// framed is the screen in three parts: a head and a foot that stay where
// they are, and a table that scrolls between them.
//
// The head says what is being looked at and the foot says how to leave, and
// both are wanted most by the reader who has scrolled furthest — which is
// exactly the reader who would have lost them.
func framed(head, body, foot []string, h, off int) []string {
	view := max(1, h-len(head)-len(foot))
	off = window(off, len(body), view)

	out := append([]string{}, head...)
	for i := off; i < len(body) && i < off+view; i++ {
		out = append(out, body[i])
	}

	for len(out) < max(h-len(foot), 0) {
		out = append(out, "")
	}

	return append(out, foot...)
}
