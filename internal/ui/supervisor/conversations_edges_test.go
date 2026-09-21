package supervisor

// The list of conversations: which row carries the cursor, which one is
// open, what is on the screen when there are more of them than rows, and
// what every door does with a cursor left past the end of the list.

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// manyConversations is a record with n of them, newest first.
func manyConversations(n int) *held {
	kept := &held{}

	for i := range n {
		id := fmt.Sprintf("c%02d", i)
		kept.lines = append(kept.lines,
			spoke(fixtureNow.Add(-time.Duration(2*n-2*i)*time.Minute), id, "operator",
				"opened "+id),
			spoke(fixtureNow.Add(-time.Duration(2*n-2*i-1)*time.Minute), id, "zeta",
				"answered "+id))
	}

	return kept
}

// TestTheChosenConversationIsOnTheScreenWhereverItIsInTheList. The list is
// two rows per conversation and taller than the body as soon as there are a
// handful of them, so the window onto it is computed from the cursor — and
// a cursor the window does not hold is a reader walking a key down against
// a list that does not move.
func TestTheChosenConversationIsOnTheScreenWhereverItIsInTheList(t *testing.T) {
	s, e := opened(t, manyConversations(14))

	convs := conversationsOf(s.all)

	for _, maxRows := range []int{4, 7, 12, 30} {
		for sel := range len(convs) {
			s.listSel = sel

			rows := s.conversationRows(maxRows, 80, e)
			if len(rows) != maxRows {
				t.Fatalf("a body of %d rows drew %d", maxRows, len(rows))
			}

			marked := 0

			for _, r := range rows {
				line := ansi.Strip(r)
				if !strings.Contains(line, "▸") {
					continue
				}

				marked++

				if !strings.Contains(line, convs[sel].title) {
					t.Errorf("in %d rows with %d chosen, the cursor is on %q", maxRows, sel, line)
				}
			}

			if marked != 1 {
				t.Errorf("in %d rows with %d chosen of %d, %d rows carry the cursor",
					maxRows, sel, len(convs), marked)
			}

			// A list longer than the body fills it: a window that ran
			// past the end of the list would draw blank rows under it
			// and say the reader had reached the bottom.
			if len(convs)*conversationRowsEach <= maxRows {
				continue
			}

			for i, r := range rows {
				if ansi.Strip(r) == "" {
					t.Fatalf("in %d rows with %d chosen of %d, row %d is blank",
						maxRows, sel, len(convs), i)
				}
			}
		}
	}
}

// TestTheOpenConversationIsTheOneMarkedAsOpen. The cursor says which one a
// key will act on and the dot says which one the thread underneath belongs
// to: two different facts, and one mark for both would make the list unable
// to say a reader is looking at one and pointing at another.
func TestTheOpenConversationIsTheOneMarkedAsOpen(t *testing.T) {
	s, e := opened(t, manyConversations(4))

	convs := conversationsOf(s.all)

	for _, open := range convs {
		s = s.openConversation(open.id, e)
		s.listSel = 0

		lit := 0

		for _, r := range s.conversationRows(30, 80, e) {
			line := ansi.Strip(r)
			if !strings.Contains(line, "●") {
				continue
			}

			lit++

			if !strings.Contains(line, open.title) {
				t.Errorf("the conversation open is %q and the one lit is %q", open.title, line)
			}
		}

		if lit != 1 {
			t.Errorf("%d conversations are marked as open, want one", lit)
		}
	}
}

// TestATitleIsCutOnlyWhenItIsLongerThanTheColumn. A title cut at exactly
// the column it fits in ends in an ellipsis that says something was left
// out when nothing was.
func TestATitleIsCutOnlyWhenItIsLongerThanTheColumn(t *testing.T) {
	for _, c := range []struct {
		n   int
		cut bool
	}{
		{titleCut - 1, false},
		{titleCut, false},
		{titleCut + 1, true},
	} {
		got := convTitle(strings.Repeat("a", c.n))
		if strings.HasSuffix(got, "…") != c.cut {
			t.Errorf("a title of %d characters came back as %q, cut=%v", c.n, got, c.cut)
		}
	}

	// And it is the first line of what was said, not the whole of it.
	if got := convTitle("primera línea\nsegunda línea"); strings.Contains(got, "segunda") {
		t.Errorf("a title of two lines is %q", got)
	}
}

// TestACursorPastTheEndOfTheListOpensAndForgetsNothing. The list is read
// from the record on every frame and a conversation can go out of it — the
// reader forgets one — so every door that reads the list by index is asked
// about the row after the last.
func TestACursorPastTheEndOfTheListOpensAndForgetsNothing(t *testing.T) {
	kept := manyConversations(3)
	s, e := opened(t, kept)

	s, _ = s.Key(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}, e)
	if !s.list {
		t.Fatal("ctrl-l did not open the conversations")
	}

	n := len(conversationsOf(s.all))
	was := s.conversation

	for _, sel := range []int{n, n + 3} {
		s.listSel = sel

		if next, _ := s.Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e); next.conversation != was {
			t.Errorf("⏎ on row %d of %d opened %q", sel, n, next.conversation)
		}

		if _, _ = s.Key(tea.KeyPressMsg{Code: 'd', Text: "d"}, e); len(kept.gone) != 0 {
			t.Fatalf("d on row %d of %d forgot %v", sel, n, kept.gone)
		}
	}
}

// TestForgettingTheLastConversationLeavesTheCursorOnTheNewLast. The cursor
// is an index into a list one shorter than it was a moment ago, and an
// index that outlives its list points at a row nobody can see.
func TestForgettingTheLastConversationLeavesTheCursorOnTheNewLast(t *testing.T) {
	kept := manyConversations(3)
	s, e := opened(t, kept)

	s.list, s.listSel = true, len(conversationsOf(s.all))-1

	gone := conversationsOf(s.all)[s.listSel].id

	next, _ := s.removeConversation(e)
	if len(kept.gone) != 1 || kept.gone[0] != gone {
		t.Fatalf("d forgot %v, want %q", kept.gone, gone)
	}

	// The record the fixture answers with has not changed, so what this
	// pins is the arithmetic: the cursor is held inside whatever is left.
	if left := len(conversationsOf(next.all)); next.listSel > max(left-1, 0) {
		t.Errorf("the cursor is on row %d of %d left", next.listSel, left)
	}
}

// TestAWindowThatCannotNameAConversationStillOpensOne. The id is the
// window's to make — it is the one that knows what a new conversation is
// called — and a screen handed no way to make one opens the nameless one
// rather than falling over.
func TestAWindowThatCannotNameAConversationStillOpensOne(t *testing.T) {
	e := world(t, &held{})
	e.NewID = nil

	s := Open(0, e)
	if s.conversation != "" {
		t.Errorf("a screen with no way to name a conversation opened %q", s.conversation)
	}

	named := world(t, &held{})

	if got := Open(0, named).conversation; got != "c-new" {
		t.Errorf("a screen that can name one opened %q, want c-new", got)
	}
}

// TestBackspaceOnAnEmptyLineTakesNothing, and on a line with one character
// in it takes that one.
func TestBackspaceOnAnEmptyLineTakesNothing(t *testing.T) {
	s, e := open(t)

	for _, code := range []rune{tea.KeyBackspace, tea.KeyDelete} {
		s.input = ""

		if next, _ := s.Key(tea.KeyPressMsg{Code: code}, e); next.input != "" {
			t.Errorf("%v on an empty line left %q", code, next.input)
		}

		s.input = "a"

		if next, _ := s.Key(tea.KeyPressMsg{Code: code}, e); next.input != "" {
			t.Errorf("%v on a line of one character left %q", code, next.input)
		}
	}
}

// TestALineNumberPastTheThreadTakesNothingBack. Picking is done against a
// thread that is read again on every frame, and the one gesture on this
// screen that cannot be undone is the one that must not act on a line that
// is not there.
func TestALineNumberPastTheThreadTakesNothingBack(t *testing.T) {
	kept := saidThree()
	s, e := opened(t, kept)

	s.picking = true

	for _, pick := range []int{len(s.lines), len(s.lines) + 5} {
		s.pick = pick

		next, out := s.retractPicked(e)
		if len(kept.back) != 0 {
			t.Fatalf("a pick of %d of %d took back %v", pick, len(s.lines), kept.back)
		}

		if next.picking || out.Said != "" {
			t.Errorf("a pick of %d left picking=%v and said %q", pick, next.picking, out.Said)
		}
	}

	// And the last line of the thread is a line.
	s.pick = len(s.lines) - 1

	if _, _ = s.retractPicked(e); len(kept.back) != 1 {
		t.Errorf("the last line of the thread took back %v", kept.back)
	}
}

// TestYesterdayIsYesterdayWhateverTheHour. A conversation at eleven last
// night is yesterday's at nine this morning, and ten hours is not what says
// so.
func TestYesterdayIsYesterdayWhateverTheHour(t *testing.T) {
	now := time.Date(2026, 9, 21, 9, 0, 0, 0, time.Local)

	for _, c := range []struct {
		then time.Time
		want int
	}{
		{now, 0},
		{time.Date(2026, 9, 21, 0, 1, 0, 0, time.Local), 0},
		{time.Date(2026, 9, 20, 23, 0, 0, 0, time.Local), 1},
		{time.Date(2026, 9, 20, 0, 0, 0, 0, time.Local), 1},
		{time.Date(2026, 9, 14, 12, 0, 0, 0, time.Local), 7},
	} {
		if got := daysBetween(c.then, now); got != c.want {
			t.Errorf("%v before %v is %d days, want %d", c.then, now, got, c.want)
		}
	}
}
