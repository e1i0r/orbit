package supervisor

// Where a click lands on this screen.
//
// It answered nothing at all until now — the window's routing returned no
// target for this screen, on the grounds that a thread is read and scrolled
// — and that was wrong about three gestures. The conversations, the offers
// over an unfinished word and picking a line to take back are all lists with
// one row chosen, and every other list in this window is clicked.
//
// The arithmetic is the drawing's own, asked the same way the keys ask it: a
// hit worked out against a different height from the one on screen is a
// click that lands a row out, which is worse than one that lands nowhere.

import (
	"github.com/e1i0r/orbit/internal/ui/point"
)

// conversationRowsEach is how many lines one conversation takes in the
// list: its title, and the line under it saying what was last said and how
// long the conversation is.
const conversationRowsEach = 2

// gutter is the two cells every row of this screen is drawn past, which is
// where the thread's own column starts.
const gutter = 2

// Hit is what the cell at (x, y) holds, in the window's own vocabulary.
//
// Anywhere inside the thread's column counts as its row: a reader who
// pointed at the end of a line meant the line, which is the rule the board's
// own rows follow. What x is read for is the column beside it — what Orbit
// knows is drawn there, it is a reading and not a list, and a click on it
// while a line was being picked would have taken back a line the pointer was
// nowhere near.
func (s State) Hit(x, y int, e Env) point.Target {
	row, ok := e.Frame.BodyRow(y)
	if !ok {
		return point.Target{}
	}

	cw, threadH := s.layout(e.Frame.Body.H, e.Frame.Body.W, e)

	// One cell past the text is the scroll rail, which belongs to the
	// thread; everything further out is the side column's.
	if s.sideFits(e.Frame.Body.W) && x > gutter+cw {
		return point.Target{}
	}

	// The offers sit between the thread and the line being typed, and they
	// own the pointer while they are up for the reason they own ↑↓: the
	// list is about the word under the cursor, and a click that reached
	// past it would answer about something the reader is not looking at.
	if at, held := s.offerAt(row, threadH, cw, e); held {
		return point.Target{Kind: point.SupervisorOffer, Pane: at}
	}

	body := row - supervisorHeadRows
	if body < 0 || body >= threadH {
		return point.Target{}
	}

	if s.list {
		return s.conversationAt(body)
	}

	if s.picking {
		return s.lineAt(body, threadH, cw, e)
	}

	return point.Target{}
}

// offerAt is which offer of the completion list a body row holds, and
// whether it holds one at all.
//
// The list is drawn under the thread and the blank row that sets it off, so
// its first row is that far down whatever the thread's height happens to be.
func (s State) offerAt(row, threadH, cw int, e Env) (int, bool) {
	drawn := s.drawCompletions(cw, e)
	if len(drawn) == 0 {
		return 0, false
	}

	first := supervisorHeadRows + threadH + 1

	at := row - first
	if at < 0 || at >= len(drawn) {
		return 0, false
	}

	// The last row of the list is the blank one under it, and the one
	// before it may be the "n more" line rather than an offer. Both are
	// the box's own chrome: a click on either is a click on nothing.
	offers := s.completions(e)

	from := 0
	if pick := min(s.pick, len(offers)-1); pick >= completionRows {
		from = pick - completionRows + 1
	}

	if from+at >= len(offers) || at >= completionRows {
		return 0, false
	}

	return from + at, true
}

// conversationAt is which conversation a row of the list holds.
//
// Two rows each, top-aligned — a list is read from its start where a thread
// is read from its end — so the row divides and a click on either line of a
// conversation is a click on that conversation.
func (s State) conversationAt(body int) point.Target {
	convs := conversationsOf(s.all)

	at := body / conversationRowsEach
	if at >= len(convs) {
		return point.Target{}
	}

	return point.Target{Kind: point.SupervisorConversation, Pane: at}
}

// lineAt is which message of the thread a body row belongs to, while a line
// is being picked.
//
// A message is several rows once it is wrapped, and starts says which row
// each one begins on: the message a row belongs to is the last one that
// begins at or above it, which is what makes a click anywhere inside a
// wrapped paragraph a click on that paragraph.
func (s State) lineAt(body, threadH, cw int, e Env) point.Target {
	rendered, starts := s.threadLines(cw, e)
	if len(rendered) == 0 || len(starts) == 0 {
		return point.Target{}
	}

	// A thread shorter than the body is drawn against the floor, with blank
	// rows above it: the first row of text is that far down. One as long as
	// the body is scrolled under it instead, and where it has been taken to
	// is the drawing's own answer rather than a second one worked out here.
	line := body + s.threadOffset(len(rendered), threadH, starts)
	if len(rendered) < threadH {
		line = body - (threadH - len(rendered))
	}

	if line < 0 || line >= len(rendered) {
		return point.Target{}
	}

	at := -1

	for i, start := range starts {
		if start <= line {
			at = i
		}
	}

	if at < 0 {
		return point.Target{}
	}

	return point.Target{Kind: point.SupervisorLine, Pane: at}
}

// Click is one click landing on what Hit named.
//
// Each of the three is the gesture its key already is, and nothing more: a
// pointer that could do something the keyboard cannot is a second set of
// rules for one screen.
func (s State) Click(t point.Target, e Env) (State, Out) {
	switch t.Kind {
	case point.SupervisorConversation:
		convs := conversationsOf(s.all)
		if t.Pane < 0 || t.Pane >= len(convs) {
			return s, Out{}
		}

		s.listSel = t.Pane

		return s.openConversation(convs[t.Pane].id, e), Out{}
	case point.SupervisorOffer:
		if t.Pane < 0 || t.Pane >= len(s.completions(e)) {
			return s, Out{}
		}

		s.pick = t.Pane

		return s.takeCompletion(e), Out{}
	case point.SupervisorLine:
		if !s.picking || t.Pane < 0 || t.Pane >= len(s.lines) {
			return s, Out{}
		}

		s.pick = t.Pane

		return s.retractPicked(e)
	}

	return s, Out{}
}
