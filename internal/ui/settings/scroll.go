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

	_, _, lines := s.placed(s.Rows(e))

	return window(s.off, lines, room(e.Frame.Body.H))
}

// headingLines is how tall a group's heading is.
const headingLines = 1

// placed is where each row starts in the table, where each group's
// heading is, and how many lines the table is: rowLines a row, and a
// heading above the first row of every group. A row of a folded group
// starts at -1, because it is not drawn; heads is -1 for a row with no
// heading above it. Scrolling and clicks both read it, so neither can
// disagree with what was drawn about where a setting is.
func (s State) placed(rows []Row) (starts, heads []int, lines int) {
	starts, heads = make([]int, len(rows)), make([]int, len(rows))

	for i := range rows {
		heads[i], starts[i] = -1, -1

		if headed(rows, i) {
			heads[i] = lines
			lines += headingLines
		}

		if !s.folded[rows[i].Group] {
			starts[i] = lines
			lines += rowLines
		}
	}

	return starts, heads, lines
}

// headed is whether row i is the first of its group, and so drawn under
// the group's heading.
func headed(rows []Row, i int) bool {
	return rows[i].Group != "" && (i == 0 || rows[i-1].Group != rows[i].Group)
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

	return s.step(d, rows, false).keepSeen(e)
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

	starts, heads, lines := s.placed(rows)
	view := room(e.Frame.Body.H)
	off := window(s.off, lines, view)

	// The blank under a row is not part of what has to be seen. A dial with
	// its name and its sentence both on the screen has been read, and
	// scrolling one line further to show the gap beneath it would move the
	// table for nothing. A heading is one line.
	from, last := starts[s.sel], starts[s.sel]+1

	if s.head != "" || from < 0 {
		from = lineOfHead(rows, heads, s.headOf(rows))
		last = from
	}

	switch {
	case from < off:
		off = from
	case last >= off+view:
		off = last - view + 1
	}

	s.off = window(atTheTop(off, view, starts, heads), lines, view)

	return s
}

// atTheTop puts the first line on show at the start of a row rather than in
// the middle of one.
//
// A view that lands mid-row draws that row's blank separator under the
// title, which reads as a gap somebody left there and spends a line of the
// table on nothing. It rounds up to the nearest place a block begins: a
// group's heading, or a row. Every row's start is one of those places, so
// the cursor's own row is never rounded off the top.
//
// Not on a screen with less room than a row, where rounding up would push
// the row the cursor is on off the top of the view it was brought into.
func atTheTop(off, view int, starts, heads []int) int {
	if view < rowLines {
		return off
	}

	best := -1

	for _, at := range append(append([]int{}, starts...), heads...) {
		if at >= off && (best < 0 || at < best) {
			best = at
		}
	}

	if best < 0 {
		return off
	}

	return best
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

// RowAt is the setting drawn on a line of the body, and whether that line
// is one of the table at all.
//
// The window worked this out for itself and knew two of the three numbers:
// the head above the table and the offset it was scrolled to, but not the
// floor the table stops at, where the line saying how to leave is drawn.
// So a click on that line, or on the blank above it, named the setting
// under the window — and a click at the columns the pills are drawn in
// turned the dial of a setting that was not on the screen.
func (s State) RowAt(line int, e Env) (int, bool) {
	on := line - headLines
	if on < 0 || on >= room(e.Frame.Body.H) {
		return 0, false
	}

	at := on + s.Off(e)
	starts, _, _ := s.placed(s.Rows(e))

	for i, start := range starts {
		if start >= 0 && at >= start && at < start+rowLines {
			return i, true
		}
	}

	// A heading, or past the table.
	return 0, false
}

// LineOf is the line of the screen row i's name is drawn on, and whether it
// is on the screen at all: RowAt the other way round, for whoever has to
// point at a setting rather than find one under the pointer.
func (s State) LineOf(i int, e Env) (int, bool) {
	starts, _, _ := s.placed(s.Rows(e))
	if i < 0 || i >= len(starts) || starts[i] < 0 {
		return 0, false
	}

	on := starts[i] - s.Off(e)
	if on < 0 || on >= room(e.Frame.Body.H) {
		return 0, false
	}

	return headLines + on, true
}

// HeadAt is the group whose heading is drawn on a line of the body, and
// whether there is one: RowAt for the lines between the rows.
func (s State) HeadAt(line int, e Env) (string, bool) {
	on := line - headLines
	if on < 0 || on >= room(e.Frame.Body.H) {
		return "", false
	}

	at := on + s.Off(e)
	rows := s.Rows(e)
	_, heads, _ := s.placed(rows)

	for i, head := range heads {
		if head >= 0 && head == at {
			return rows[i].Group, true
		}
	}

	return "", false
}

// headOf is the group the cursor is in: the heading it stands on, or the
// group of its row.
func (s State) headOf(rows []Row) string {
	if s.head != "" || s.sel < 0 || s.sel >= len(rows) {
		return s.head
	}

	return rows[s.sel].Group
}

// lineOfHead is the line a group's heading is drawn on.
func lineOfHead(rows []Row, heads []int, group string) int {
	for i, head := range heads {
		if head >= 0 && rows[i].Group == group {
			return head
		}
	}

	return 0
}
