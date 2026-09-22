package repos

// The list is longer than the screen as soon as somebody works in a handful
// of repositories, and it was drawn whole into a body that then cut it.
//
// What that cost was not only the rows past the bottom: the cursor walked
// down into them, so ⏎ filtered the board to a repository the reader could
// not see, and nothing on the screen said which one it was.

// What stands above the list and below it, and never scrolls: the blank
// row, the title, the sentence under it and the blank that sets the list
// off; then the blank row and the line of keys at the foot.
//
// The head says what is being looked at and the foot says how to leave, and
// both are wanted most by the reader who has scrolled furthest — which is
// exactly the reader who would have lost them.
const (
	headRows = 4
	waysRows = 2
)

// room is how many rows of the list the screen has left over for it.
func room(h int) int { return max(1, h-headRows-waysRows) }

// window is the first row of the list on show, held inside a list that may
// have got shorter since it was set: a task finishing takes a repository
// nothing is filed under off the board, and a view parked past the end
// draws blank rows with nothing on the screen saying why.
func window(off, rows, view int) int {
	return min(max(off, 0), max(0, rows-view))
}

// Off is the first row of the list on show.
//
// It is a door because a click is measured from it: the row under the
// pointer is the line it landed on plus however far the list has been
// scrolled, and a window that did not add it would filter to whichever
// repository used to be there.
func (s State) Off(e Env) int {
	if e.Frame.Body.H <= 0 {
		return 0
	}

	return window(s.off, len(collect(e)), room(e.Frame.Body.H))
}

// Scroll moves the cursor by d and brings the list with it. It is what the
// wheel does: a list with one row chosen moves the choice, which is what
// every other list in this window does under the same notch.
func (s State) Scroll(d int, e Env) State {
	repos := collect(e)
	if len(repos) == 0 {
		return s
	}

	s.sel = min(max(s.sel+d, 0), len(repos)-1)

	return s.keepSeen(e)
}

// keepSeen brings the row the cursor is on back onto the screen.
//
// A window that has not been sized yet holds nothing, and the height this
// would scroll against is not the height View will be given.
func (s State) keepSeen(e Env) State {
	repos := collect(e)
	if len(repos) == 0 || s.sel < 0 || s.sel >= len(repos) || e.Frame.Body.H <= 0 {
		s.off = 0

		return s
	}

	view := room(e.Frame.Body.H)

	off := window(s.off, len(repos), view)
	switch {
	case s.sel < off:
		off = s.sel
	case s.sel >= off+view:
		off = s.sel - view + 1
	}

	s.off = window(off, len(repos), view)

	return s
}

// framed is the screen in three parts: a head and a foot that stay where
// they are, and a list that scrolls between them.
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

// RowAt is the repository drawn on a line of the body, and whether that
// line is one of the list at all.
//
// The window used to work this out for itself — the line, less the rows of
// the head, plus the offset — and it left out the one thing the drawing
// knows: the list stops short of the floor, where the ways out are. So a
// click on the blank above them, or on the line of keys itself, named the
// repository just under the window and filtered the board to a checkout
// the reader had never seen. That is the failure this file was written to
// stop, arrived at by the mouse instead of the cursor.
func (s State) RowAt(line int, e Env) (Item, bool) {
	list := collect(e)

	on := line - headRows
	if on < 0 || on >= room(e.Frame.Body.H) {
		return Item{}, false
	}

	at := on + s.Off(e)
	if at < 0 || at >= len(list) {
		return Item{}, false
	}

	return list[at], true
}
