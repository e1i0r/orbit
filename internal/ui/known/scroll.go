package known

// The window: which part of a list longer than the screen is on it, and what
// is under the pointer.
//
// A list that cannot scroll is a list with a ceiling nobody declared. Orbit
// learns four ways and keeps everything it is told, so this screen is one of
// the few that grows on its own — and the rules past the bottom of it were
// exactly the ones nobody could find.

import (
	"github.com/e1i0r/orbit/internal/ui/point"
)

// span is where one row the cursor can be on was drawn: the line it starts
// at, and how many lines it took. A rule is as tall as its sentence wraps,
// so neither the scrolling nor a click can assume one row is one line.
type span struct{ from, rows int }

// last is the bottom line of the row.
func (sp span) last() int { return sp.from + max(sp.rows, 1) - 1 }

// holds is whether a line of the body is part of this row.
func (sp span) holds(line int) bool { return line >= sp.from && line <= sp.last() }

// framed is a screen in three parts: a head and a foot that stay where they
// are, and a body that scrolls between them.
//
// The head says what is being looked at and the foot what the keys do, and
// both are needed most by the reader who has scrolled furthest — which is
// exactly the reader who would have lost them.
func framed(head, body, foot []string, h, off int) []string {
	room := max(1, h-len(head)-len(foot))
	off = min(max(off, 0), max(0, len(body)-room))

	out := append([]string{}, head...)
	for i := off; i < len(body) && i < off+room; i++ {
		out = append(out, body[i])
	}

	for len(out) < max(h-len(foot), 0) {
		out = append(out, "")
	}

	return append(out, foot...)
}

// deepEnough brings one stretch of a body on screen, moving as little as it
// can: it is what keeps the row being typed into visible as the form is
// walked, and what the arrow keys move on a rule's own screen.
func deepEnough(off, from, rows, room, total int) int {
	switch {
	case rows >= room || from < off:
		off = from
	case from+rows > off+room:
		off = from + rows - room
	}

	return min(max(off, 0), max(0, total-room))
}

// content is how wide the screen draws, which is the window less its margins
// and capped so that a sentence does not stretch across a monitor.
func content(w int) int { return max(min(w-4, 110), 24) }

// window is the first line of the body on show, held inside the list it is
// scrolling.
//
// Clamped here rather than where it is set, because the list gets shorter on
// its own: a sentence kept leaves the tray, a rule turned off shortens no
// row but a repository emptied loses a heading. A view parked past the end
// of a list that shrank is a screen of blank rows and no way to tell why.
func (s State) window(lines, view int) int {
	return min(max(s.offset, 0), max(0, lines-view))
}

// view is how many rows of the list the screen has room for: what the head
// and the foot did not take.
func (s State) view(cw, h int, e Env) int {
	return max(1, h-len(s.head(cw, e))-len(s.foot(cw, e)))
}

// keepSeen moves the list as little as it can to keep the row the cursor is
// on in sight. The reader is moving a cursor; the scrolling is this screen's
// business rather than theirs.
func (s State) keepSeen(e Env) State {
	cw, h := content(e.Frame.Body.W), e.Frame.Body.H
	if h <= 0 || cw <= 0 {
		return s
	}

	body, at := s.body(cw, e)
	if s.sel < 0 || s.sel >= len(at) {
		s.offset = 0
		return s
	}

	room := s.view(cw, h, e)
	row := at[s.sel]
	off := s.window(len(body), room)

	switch {
	case row.rows >= room:
		// A row taller than the room it has is shown from its top. Its
		// first line is the one with the columns on it, and the bottom of
		// a rule with its head off the screen says nothing about which
		// rule it is.
		off = row.from
	case row.from < off:
		// The two rows above come on screen with it: "orbit · travels with
		// the repository" is what says which project the rule is about, and
		// the row of column names under it is what says which field is
		// which. A rule alone at the top of the screen says neither.
		off = max(0, row.from-2)
	case row.last() >= off+room:
		off = row.last() - room + 1
	}

	s.offset = min(max(off, 0), max(0, len(body)-room))

	return s
}

// move walks the cursor, and brings the list with it. On a screen opened
// over the list it moves that screen instead: one gesture, whatever is up.
func (s State) move(d int, e Env) State {
	if s.reading || s.editing {
		s.deep = max(s.deep+d, 0)

		return s
	}

	s.sel = min(max(s.sel+d, 0), s.last())

	return s.keepSeen(e)
}

// step is one press of an arrow: move does it, except that off either end
// of the list it comes round to the other. A screen being read is text,
// and text stops where it stops.
func (s State) step(d int, e Env) State {
	if s.reading || s.editing {
		return s.move(d, e)
	}

	n := s.last() + 1
	s.sel = (s.sel + d + n) % n

	return s.keepSeen(e)
}

// hit is what the screen has at that cell.
//
// Only the rows answer. The headings, the column names and the blank rows
// between two groups are furniture, and a click that landed on furniture and
// moved the cursor to whatever was nearest is the gesture a reader learns
// not to trust.
func (s State) hit(x, y int, e Env) point.Target {
	line, ok := e.Frame.BodyRow(y)
	if !ok {
		return point.Target{}
	}

	if r, up := s.picked(e); up {
		// The list's own rows, counted from the line under its top border.
		at := s.pickAt(line, r, content(e.Frame.Body.W), e)
		if at < 0 {
			return point.Target{}
		}

		return point.Target{Kind: point.KnowledgePick, Pane: at}
	}

	if s.reading || s.editing {
		// A screen opened over the list owns the pointer while it is up,
		// the way it owns the keyboard. A click that reached the list
		// behind it would move a cursor nobody can see.
		return point.Target{}
	}

	cw := content(e.Frame.Body.W)

	body, at := s.body(cw, e)

	// The head is drawn above the body and the window is scrolled under it,
	// so a cell's line in the body is where it was drawn plus how far down
	// the list has been taken — and only where the list is drawn at all.
	// The foot is the line that says what the keys do: counted from the
	// head alone, a click on it landed on the rule below the window.
	view := s.view(cw, e.Frame.Body.H, e)

	on := line - len(s.head(cw, e))
	if on < 0 || on >= view {
		return point.Target{}
	}

	want := on + s.window(len(body), view)

	for i, row := range at {
		if row.holds(want) {
			return point.Target{Kind: point.KnowledgeRow, Pane: i}
		}
	}

	return point.Target{}
}

// pointAt puts the cursor on the row that was clicked, and says whether it
// was already there.
//
// A row that is not the cursor's takes one click to become it and a second
// to open, which is the two-step every list in the window already has. It is
// not a double-click: a double-click is a timer, and a timer means the same
// two clicks do different things depending on how fast the reader is.
func (s State) pointAt(i int, e Env) (State, bool) {
	if i < 0 || i > s.last() {
		return s, false
	}

	if s.sel == i {
		return s, true
	}

	s.sel = i

	return s.keepSeen(e), false
}
