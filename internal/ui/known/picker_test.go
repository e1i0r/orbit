package known

// Choosing one of a row's options from a list.
//
// None of this was tested: every function of picker.go read zero, and it is
// the whole of how a rule's place and its check are chosen now — the rows
// that used to be empty lines somebody had to spell a path into.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// writing is the screen with a fresh rule open for filling in, the cursor on
// the row that says where the rule applies: the whole checkout, each of its
// top folders, and whatever somebody typed that is none of those.
//
// That row and not another because it is the one whose answer is the value:
// picking a place puts the path in the field. The row that says what a rule
// does is a shortcut over the check and is tested where the check is.
func writing(t *testing.T) (State, Env) {
	t.Helper()

	s, e := onScreen(t, knowledge.Rule{
		ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human, Phrase: "the fuzz tests hang",
	})
	e.Replace = func(knowledge.Rule, knowledge.Rule) error { return nil }
	e.Places = func(string) []string { return []string{"internal/db", "internal/ui"} }

	s, _ = s.Key(press("n"), e)
	if !s.editing {
		t.Fatal("n did not open a rule to write")
	}

	return onRow(t, s, e, rowWhere), e
}

// onRow tabs until the cursor is on that row, so the walk is the one a
// reader takes rather than a field set from outside.
func onRow(t *testing.T, s State, e Env, which int) State {
	t.Helper()

	for range len(s.rows2(e)) {
		if s.field == which {
			return s
		}

		s, _ = s.Key(press("tab"), e)
	}

	t.Fatalf("tab never reached row %d", which)

	return s
}

// TestARowWithAListOpensItRatherThanSaving. Enter is the form's one obvious
// answer, and on a row holding a list the obvious answer is the list: a
// lit word beside a grey word does not read as a choice.
func TestARowWithAListOpensItRatherThanSaving(t *testing.T) {
	s, e := writing(t)

	s, _ = s.Key(press("enter"), e)

	if !s.picking {
		t.Fatal("enter on a row with a list did not open it")
	}

	if !s.editing {
		t.Error("opening the list closed the form")
	}

	r, up := s.picked(e)
	if !up {
		t.Fatal("the open list belongs to no row")
	}

	if len(r.options) < pickFrom {
		t.Errorf("the open list holds %d options, want at least %d", len(r.options), pickFrom)
	}
}

// TestTheListOpensOnWhatTheRowIsHolding. A list that opened at the top
// would make changing nothing cost as much as changing something, and the
// reader has to find their own answer again every time they look.
func TestTheListOpensOnWhatTheRowIsHolding(t *testing.T) {
	s, e := writing(t)

	// Take the second option — the first folder — and open the list again.
	s, _ = s.Key(press("enter"), e)
	s, _ = s.Key(press("down"), e)
	s, _ = s.Key(press("enter"), e)

	if s.picking {
		t.Fatal("enter on an option left the list open")
	}

	r, _ := func() (aRow, bool) {
		open, _ := s.Key(press("enter"), e)

		return open.picked(e)
	}()

	if got := s.held(r); got != r.options[1].value {
		t.Fatalf("the row holds %q, want the option that was chosen", got)
	}

	s, _ = s.Key(press("enter"), e)

	if s.pick != 1 {
		t.Errorf("the list opened on option %d, want the one the row is holding", s.pick)
	}
}

// TestTheCursorStopsAtBothEndsOfTheList. It is a list with one row chosen,
// so it answers what every such list in this window answers — and running
// off either end is how a cursor ends up pointing at nothing.
func TestTheCursorStopsAtBothEndsOfTheList(t *testing.T) {
	s, e := writing(t)
	s, _ = s.Key(press("enter"), e)

	r, _ := s.picked(e)

	for range len(r.options) + 3 {
		s, _ = s.Key(press("down"), e)
	}

	if s.pick != len(r.options)-1 {
		t.Errorf("the cursor ran to %d, want it to stop at %d", s.pick, len(r.options)-1)
	}

	for range len(r.options) + 3 {
		s, _ = s.Key(press("up"), e)
	}

	if s.pick != 0 {
		t.Errorf("the cursor ran to %d, want it to stop at the top", s.pick)
	}
}

// TestTheOtherTwoKeysMoveTheCursorToo. j and k are what the rest of this
// window takes, and a list that only answered the arrows would be the one
// place a reader's hands stop working.
func TestTheOtherTwoKeysMoveTheCursorToo(t *testing.T) {
	s, e := writing(t)
	s, _ = s.Key(press("enter"), e)

	s, _ = s.Key(press("j"), e)
	if s.pick != 1 {
		t.Errorf("j left the cursor at %d, want 1", s.pick)
	}

	s, _ = s.Key(press("k"), e)
	if s.pick != 0 {
		t.Errorf("k left the cursor at %d, want 0", s.pick)
	}

	// A key the list has nothing to say about leaves it alone rather than
	// closing it or typing into the form behind it.
	s, _ = s.Key(press("z"), e)
	if !s.picking || s.pick != 0 {
		t.Errorf("an unknown key left picking=%v at %d", s.picking, s.pick)
	}
}

// TestLeavingTheListChangesNothing. Escape is how a reader gets out of
// something they opened by accident, and a list that took the cursor's
// option on the way out would make looking cost a change.
func TestLeavingTheListChangesNothing(t *testing.T) {
	s, e := writing(t)

	r, _ := func() (aRow, bool) {
		s, _ := s.Key(press("enter"), e)

		return s.picked(e)
	}()

	before := s.held(r)

	s, _ = s.Key(press("enter"), e)
	s, _ = s.Key(press("down"), e)
	s, _ = s.Key(press("esc"), e)

	if s.picking {
		t.Error("escape left the list open")
	}

	if !s.editing {
		t.Error("escape closed the form as well as the list")
	}

	if got := s.held(r); got != before {
		t.Errorf("leaving the list changed the row from %q to %q", before, got)
	}
}

// TestTheListSaysWhatEachAnswerMeans. A column of names with the reasons
// beside them is scanned; a list of sentences is read word by word, and what
// somebody opening it wants is the one they already had in mind.
func TestTheListSaysWhatEachAnswerMeans(t *testing.T) {
	s, e := writing(t)
	s, _ = s.Key(press("enter"), e)

	drawn := drawnKnowledge(t, s, e)

	r, _ := s.picked(e)
	for _, one := range r.options {
		if !strings.Contains(drawn, one.label) {
			t.Errorf("the list does not offer %q:\n%s", one.label, drawn)
		}

		if one.note != "" && !strings.Contains(drawn, one.note) {
			t.Errorf("the list does not say what %q means:\n%s", one.label, drawn)
		}
	}

	// And it says how to leave, because a box with no way out written on it
	// is a reader pressing keys to find one.
	for _, way := range []string{"choose it", "leave it"} {
		if !strings.Contains(drawn, way) {
			t.Errorf("the list does not say how to %s:\n%s", way, drawn)
		}
	}
}

// TestAClosedRowSaysThereIsAListBehindIt. A row that always says "▾ 2 to
// choose from" needs working out once; one that shows nothing is a choice
// nobody knows they have.
func TestAClosedRowSaysThereIsAListBehindIt(t *testing.T) {
	s, e := writing(t)

	drawn := drawnKnowledge(t, s, e)
	if !strings.Contains(drawn, "to choose from") {
		t.Errorf("a row with a list behind it does not say so:\n%s", drawn)
	}

	// A row offering one answer offers no choice, and saying so would be a
	// hint about a list that does not open.
	one := aRow{options: []option{{label: "everywhere"}}}
	if hint := s.pickHint(one, e); hint != "" {
		t.Errorf("a row with one answer says %q, want nothing", hint)
	}
}

// TestThePointerTakesTheOptionItLandedOn, which is the same gesture enter is
// on the row the cursor is already on.
func TestThePointerTakesTheOptionItLandedOn(t *testing.T) {
	s, e := writing(t)
	s, _ = s.Key(press("enter"), e)

	r, _ := s.picked(e)

	s = s.Pick(len(r.options)-1, e)

	if s.picking {
		t.Error("taking an option with the pointer left the list open")
	}

	if got := s.held(r); got != r.options[len(r.options)-1].value {
		t.Errorf("the row holds %q, want the option the pointer landed on", got)
	}
}

// TestAPointerOutsideTheListTakesNothing. A click on the box's own chrome,
// or past the last option, is a click on nothing — and taking the cursor's
// answer instead would be the window deciding for the reader.
func TestAPointerOutsideTheListTakesNothing(t *testing.T) {
	s, e := writing(t)
	s, _ = s.Key(press("enter"), e)

	r, _ := s.picked(e)
	before := s.held(r)

	for _, at := range []int{-1, len(r.options), len(r.options) + 5} {
		next := s.Pick(at, e)

		if !next.picking {
			t.Errorf("a pointer at %d closed the list", at)
		}

		if got := next.held(r); got != before {
			t.Errorf("a pointer at %d changed the row to %q", at, got)
		}
	}

	// And with no list open at all there is nothing to take.
	shut, _ := s.Key(press("esc"), e)
	if got := shut.Pick(0, e); got.picking {
		t.Error("the pointer opened a list that was shut")
	}
}

// TestWhichLineOfTheBodyIsWhichOption. The rows of the box are the only
// lines a click means anything on; the border and the keys below it are the
// box's own chrome.
func TestWhichLineOfTheBodyIsWhichOption(t *testing.T) {
	s, e := writing(t)
	s, _ = s.Key(press("enter"), e)

	r, _ := s.picked(e)
	cw := content(e.Frame.Body.W)
	first := len(s.formAbove(cw, e)) + 1

	if got := s.pickAt(first, r, cw, e); got != 0 {
		t.Errorf("the first line of the box is option %d, want the first", got)
	}

	if got := s.pickAt(first-1, r, cw, e); got != -1 {
		t.Errorf("the box's top border is option %d, want none", got)
	}

	// Past the last option the box is still drawn and still means nothing.
	if got := s.pickAt(first+len(r.options), r, cw, e); got != -1 {
		t.Errorf("a line past the last option is option %d, want none", got)
	}

	if got := s.pickAt(first+pickRows, r, cw, e); got != -1 {
		t.Errorf("a line past the box is option %d, want none", got)
	}
}
