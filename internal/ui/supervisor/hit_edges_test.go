package supervisor

// The edges of where a click lands: the cell the thread ends at, the row
// the body ends at, and the row past the last item of every list on this
// screen.
//
// A hit-tester is read one coordinate at a time and written as a pair of
// sums, so the cell it is wrong about is never the one a test happens to
// point at. These walk the coordinate instead: every row of the body, every
// item of the list, and the cells either side of each boundary.

import (
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/view"
)

// TestTheRailBelongsToTheThreadAndTheCellPastItDoesNot. One cell past the
// text is the scroll rail, which is the thread's; everything further out is
// the side column's, and a click there while a line is being picked would
// take back a line the pointer was nowhere near.
func TestTheRailBelongsToTheThreadAndTheCellPastItDoesNot(t *testing.T) {
	e := wideWorld(t, saidThree())
	e.Knows = func() []knowledge.Rule {
		return []knowledge.Rule{{
			ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
			Source: knowledge.Human, Phrase: "never log a card number",
		}}
	}

	s := Open(0, e)

	s, _ = s.Key(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}, e)
	if !s.picking {
		t.Fatal("ctrl-r did not start picking a line")
	}

	if !s.sideFits(e.Frame.Body.W) {
		t.Fatalf("a body of %d columns does not fit the side, and this test is about it",
			e.Frame.Body.W)
	}

	cw, threadH := s.layout(e.Frame.Body.H, e.Frame.Body.W, e)

	rendered, starts := s.threadLines(cw, e)
	row := at(e, supervisorHeadRows+(threadH-len(rendered))+starts[0])

	for _, c := range []struct {
		x    int
		want point.Kind
	}{
		{gutter, point.SupervisorLine},
		{gutter + cw - 1, point.SupervisorLine},
		// The rail itself.
		{gutter + cw, point.SupervisorLine},
		// The first cell of the column beside it.
		{gutter + cw + 1, point.None},
		{e.Frame.Body.W - 1, point.None},
	} {
		if got := s.Hit(c.x, row, e); got.Kind != c.want {
			t.Errorf("the cell at column %d of a thread %d wide is %v, want %v",
				c.x, cw, got.Kind, c.want)
		}
	}
}

// TestEveryConversationOwnsItsOwnTwoRows, and the rows under the last of
// them own nothing. The list is two rows per conversation — its title, and
// what was last said under it — so a divide that is out by one opens the
// conversation under the one that was pointed at.
func TestEveryConversationOwnsItsOwnTwoRows(t *testing.T) {
	// Both shapes: a list the body has room to spare under, where the rows
	// past the last conversation are the ones that must answer nothing,
	// and one longer than the body, where the last row drawn is a
	// conversation and the row under it is off the screen.
	for _, conversations := range []int{3, 14} {
		kept := &held{}

		for i := range conversations {
			id := fmt.Sprintf("c%02d", i)
			kept.lines = append(kept.lines,
				spoke(fixtureNow.Add(-time.Duration(2*conversations-2*i)*time.Minute),
					id, "operator", "opened "+id),
				spoke(fixtureNow.Add(-time.Duration(2*conversations-2*i-1)*time.Minute),
					id, "zeta", "answered "+id))
		}

		s, e := opened(t, kept)

		s, _ = s.Key(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}, e)
		if !s.list {
			t.Fatal("ctrl-l did not open the conversations")
		}

		convs := conversationsOf(s.all)
		if len(convs) != conversations {
			t.Fatalf("the record has %d conversations, want %d", len(convs), conversations)
		}

		_, threadH := s.layout(e.Frame.Body.H, e.Frame.Body.W, e)

		for body := range threadH {
			got := s.Hit(gutter, at(e, supervisorHeadRows+body), e)

			want := body / conversationRowsEach
			if want >= len(convs) {
				if got.Kind != point.None {
					t.Errorf("with %d conversations, body row %d is past the last and reads as %v %d",
						conversations, body, got.Kind, got.Pane)
				}

				continue
			}

			if got.Kind != point.SupervisorConversation || got.Pane != want {
				t.Errorf("with %d conversations, body row %d is %v %d, want conversation %d",
					conversations, body, got.Kind, got.Pane, want)
			}
		}
	}
}

// TestTheRowsEitherSideOfTheBodyAreNothing. The head above the thread and
// whatever is under it are the window's, not this screen's: a body row
// counted from the wrong end answers about a row the reader cannot see.
func TestTheRowsEitherSideOfTheBodyAreNothing(t *testing.T) {
	s, e := opened(t, twoThreads())

	s, _ = s.Key(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}, e)
	if !s.list {
		t.Fatal("ctrl-l did not open the conversations")
	}

	_, threadH := s.layout(e.Frame.Body.H, e.Frame.Body.W, e)

	if got := s.Hit(gutter, at(e, supervisorHeadRows), e); got.Kind != point.SupervisorConversation {
		t.Errorf("the first row of the body is %v, want the first conversation", got.Kind)
	}

	for _, body := range []int{-1, threadH, threadH + 1} {
		row := supervisorHeadRows + body
		if row < 0 {
			continue
		}

		if got := s.Hit(gutter, at(e, row), e); got.Kind != point.None {
			t.Errorf("body row %d of %d is %v, want nothing", body, threadH, got.Kind)
		}
	}
}

// TestAnOfferIsTheOfferItIsDrawnAs, however far down the list has been
// walked. The list shows six at a time and the keyboard can walk past them,
// so the row pointed at is an offset into what is drawn and the offer taken
// is an index into the whole of it — two counts that a click is the only
// gesture asking to agree.
func TestAnOfferIsTheOfferItIsDrawnAs(t *testing.T) {
	s, e := open(t)
	s = s.Type("/")

	offers := s.completions(e)
	if len(offers) <= completionRows {
		t.Skipf("the gestures are %d, and this needs more than the %d drawn at once",
			len(offers), completionRows)
	}

	_, threadH := s.layout(e.Frame.Body.H, e.Frame.Body.W, e)
	first := supervisorHeadRows + threadH + 1

	for _, pick := range []int{0, 1, completionRows - 1, completionRows, len(offers) - 1} {
		s.pick = pick

		from := 0
		if pick >= completionRows {
			from = pick - completionRows + 1
		}

		for row := range completionRows {
			got := s.Hit(gutter, at(e, first+row), e)

			want := from + row
			if want >= len(offers) {
				if got.Kind == point.SupervisorOffer {
					t.Errorf("with %d picked, row %d of the list is offer %d of %d",
						pick, row, got.Pane, len(offers))
				}

				continue
			}

			if got.Kind != point.SupervisorOffer || got.Pane != want {
				t.Errorf("with %d picked, row %d of the list is %v %d, want offer %d",
					pick, row, got.Kind, got.Pane, want)
			}
		}

		// And the row under the last drawn one is the box's own chrome.
		if got := s.Hit(gutter, at(e, first+completionRows), e); got.Kind == point.SupervisorOffer {
			t.Errorf("with %d picked, the row under the list is offer %d", pick, got.Pane)
		}
	}
}

// TestClickingARowThatIsNotThereChangesNothing. Every target this screen
// answers carries an index into a list that can have got shorter since it
// was drawn — a line taken back, a conversation forgotten — so each of the
// three doors is asked about the row before the first and the row after the
// last of its own list, which are different lists and different lengths.
func TestClickingARowThatIsNotThereChangesNothing(t *testing.T) {
	kept := saidThree()
	s, e := opened(t, kept)

	s = s.Type("/")

	s, _ = s.Key(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}, e)
	if !s.picking {
		t.Fatal("ctrl-r did not start picking a line")
	}

	for _, c := range []struct {
		kind point.Kind
		n    int
	}{
		{point.SupervisorConversation, len(conversationsOf(s.all))},
		{point.SupervisorOffer, len(s.completions(e))},
		{point.SupervisorLine, len(s.lines)},
	} {
		for _, pane := range []int{-1, c.n, c.n + 9} {
			next, _ := s.Click(point.Target{Kind: c.kind, Pane: pane}, e)

			if len(kept.back) != 0 {
				t.Fatalf("%v %d of %d took a line back", c.kind, pane, c.n)
			}

			if next.pick != s.pick || next.listSel != s.listSel || next.input != s.input {
				t.Errorf("%v %d of %d moved the screen", c.kind, pane, c.n)
			}
		}
	}

	// And the first row of each list is a row: the guard stops at what is
	// not there rather than at what is.
	fresh, e2 := opened(t, saidThree())

	listed, _ := fresh.Key(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}, e2)
	if !listed.list {
		t.Fatal("ctrl-l did not open the conversations")
	}

	first := point.Target{Kind: point.SupervisorConversation, Pane: 0}

	if next, _ := listed.Click(first, e2); next.list {
		t.Error("clicking the first conversation left the list open")
	}

	offer := point.Target{Kind: point.SupervisorOffer, Pane: 0}

	if next, _ := s.Click(offer, e); next.input == s.input {
		t.Errorf("taking the first offer left the line as %q", next.input)
	}

	if _, _ = s.Click(point.Target{Kind: point.SupervisorLine, Pane: 0}, e); len(kept.back) == 0 {
		t.Error("clicking the first line of the thread took nothing back")
	}
}

// TestAPickLeftPastTheEndOfTheOffersIsHeldInside. The list is narrowed by
// every character typed into the word under it, so the offer the keyboard
// walked to can be past the end of the list by the time the next frame is
// drawn — and the window the list is drawn through is computed from it.
func TestAPickLeftPastTheEndOfTheOffersIsHeldInside(t *testing.T) {
	s, e := open(t)
	s = s.Type("/")

	offers := s.completions(e)
	if len(offers) <= completionRows {
		t.Skipf("the gestures are %d, and this needs more than the %d drawn at once",
			len(offers), completionRows)
	}

	_, threadH := s.layout(e.Frame.Body.H, e.Frame.Body.W, e)
	first := supervisorHeadRows + threadH + 1

	held := s
	held.pick = len(offers)

	beyond := s
	beyond.pick = len(offers) + 7

	last := s
	last.pick = len(offers) - 1

	for row := range completionRows {
		want := last.Hit(gutter, at(e, first+row), e)

		for _, s := range []State{held, beyond} {
			if got := s.Hit(gutter, at(e, first+row), e); got != want {
				t.Errorf("with a pick of %d, row %d of the list is %v %d, want %v %d",
					s.pick, row, got.Kind, got.Pane, want.Kind, want.Pane)
			}
		}
	}
}

// TestALineIsTheMessageItBelongsTo, whether the thread is shorter than the
// body it is drawn in or longer than it. A thread that does not fill the
// body is drawn against the floor with blank rows above it, and one that
// fills it is scrolled under it — two different sums from a row to a line,
// and the shorter one is what a new conversation is.
func TestALineIsTheMessageItBelongsTo(t *testing.T) {
	for _, c := range []struct {
		name string
		said []view.SupervisorLine
	}{
		{"a thread shorter than the body", saidThree().lines},
		{"a thread longer than the body", longThread(40)},
	} {
		kept := &held{lines: c.said}

		s, e := opened(t, kept)

		s, _ = s.Key(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}, e)
		if !s.picking {
			t.Fatalf("%s: ctrl-r did not start picking a line", c.name)
		}

		cw, threadH := s.layout(e.Frame.Body.H, e.Frame.Body.W, e)

		rendered, starts := s.threadLines(cw, e)

		for i, start := range starts {
			row := start
			if len(rendered) < threadH {
				row += threadH - len(rendered)
			} else {
				row -= s.threadOffset(len(rendered), threadH, starts)
			}

			if row < 0 || row >= threadH {
				continue
			}

			got := s.Hit(gutter, at(e, supervisorHeadRows+row), e)
			if got.Kind != point.SupervisorLine || got.Pane != i {
				t.Errorf("%s: the row message %d starts on is %v %d", c.name, i, got.Kind, got.Pane)
			}
		}
	}
}
