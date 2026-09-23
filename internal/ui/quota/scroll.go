package quota

// The reading is longer than the screen as soon as a build can run four
// engines: each one is a blank row, its name, and a line per window it has.
// It was drawn whole into a body that then cut it, and there was nothing to
// press — no cursor, no scroll, no wheel — so the engines past the bottom
// were a reading nobody could get to.

// What stands above the reading and below it, and never scrolls.
//
// The head says what is being looked at and the foot says how to leave, and
// both are wanted most by the reader who has scrolled furthest.
const (
	headRows = 3
	waysRows = 2
)

// room is how many rows of the reading the screen has left over for it.
func room(h int) int { return max(1, h-headRows-waysRows) }

// window is the first row on show, held inside a reading that may have got
// shorter since it was set: an engine that stops answering takes its
// windows off the screen, and a view parked past the end draws blank rows
// with nothing saying why.
func window(off, rows, view int) int {
	return min(max(off, 0), max(0, rows-view))
}

// State is how far down the reading the screen has been taken.
//
// The screen holds this and nothing else: there is still nothing on it to
// choose, which is why it has an offset and not a cursor.
type State struct {
	off int
}

// Open is the screen as it comes up: at the top of the reading.
func Open() State { return State{} }

// Scroll moves the reading by d rows, which is what both the arrows and the
// wheel do — there is nothing here to choose, so what moves is the page.
func (s State) Scroll(d int, e Env) State {
	s.off += d

	return s
}

// Off is the first row of the reading on show, for a given body.
func (s State) Off(h int, e Env) int {
	if h <= 0 {
		return 0
	}

	return window(s.off, len(readingRows(0, e)), room(h))
}

// framed is the screen in three parts: a head and a foot that stay where
// they are, and the reading that scrolls between them.
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
