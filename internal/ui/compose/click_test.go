package compose

// The compose form under the pointer. Every dial on it is a row of pills,
// and a pill nobody can land on is a choice the form does not really offer:
// these tests reach each one the way a reader does, by the cell it was drawn
// in rather than by a column written into the test.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// formRow is the screen row a labelled line of the form was drawn on.
func formRow(t *testing.T, s State, e Env, label string) int {
	t.Helper()

	y := rowOf(screenRows(s, e), label)
	if y < 0 {
		t.Fatalf("the form has no row saying %q", label)
	}

	return y
}

// pointAt is the first cell of row y the window answers for with the target
// asked for. A dial whose pill is not reachable fails here rather than in an
// assertion further down, because the two are different faults: one is a
// pill drawn where nothing is listening, the other is a click that did the
// wrong thing.
func pointAt(t *testing.T, s State, e Env, y int, kind point.Kind, pane int) int {
	t.Helper()

	for x := range e.Frame.Body.W {
		if at := s.Hit(x, y, e); at.Kind == kind && at.Pane == pane {
			return x
		}
	}

	t.Fatalf("no cell of row %d answers with kind %d, pane %d", y, kind, pane)

	return -1
}

// clickAt presses and releases on the cell pointAt found.
func clickAt(t *testing.T, s State, e Env, y int, kind point.Kind, pane int) State {
	t.Helper()

	x := pointAt(t, s, e, y, kind, pane)

	next, _ := s.Click(s.Hit(x, y, e), e)

	return next
}

// TestEveryDialOfTheFormIsChosenByPointingAtIt. One row of pills, and the
// pointer is the only way to reach one of them that does not involve
// counting keystrokes.
func TestEveryDialOfTheFormIsChosenByPointingAtIt(t *testing.T) {
	s, e := form(t)

	s = clickAt(t, s, e, formRow(t, s, e, "flow:"), point.ComposeFlowChoice, 2)
	if s.flowIdx != 2 {
		t.Errorf("the third flow pill left flowIdx=%d, want 2", s.flowIdx)
	}
}

// TestPointingBesideThePillsPutsTheCursorOnThatRow. The empty half of a dial
// row is still that dial: a reader who lands there has said which field they
// are on, and the keyboard takes it from there.
func TestPointingBesideThePillsPutsTheCursorOnThatRow(t *testing.T) {
	s, e := form(t)

	y := formRow(t, s, e, "flow:")

	at := s.Hit(e.Frame.Body.W-2, y, e)
	if at.Kind != point.ComposeField || at.Pane != composeFlow {
		t.Fatalf("the far end of the flow row is kind %d pane %d, want the flow field", at.Kind, at.Pane)
	}

	s, _ = s.Click(at, e)
	if s.field != composeFlow {
		t.Errorf("the cursor is on field %d, want the flow one", s.field)
	}
}

// TestTheFlowRowOffersMoreThanFlows. Two things sit at the end of it — the
// button that writes a new flow, and, one row down, the summary of the flow
// that is chosen. Both open a screen, and neither is a flow.
func TestTheFlowRowOffersMoreThanFlows(t *testing.T) {
	s, e := form(t)

	y := formRow(t, s, e, "flow:")

	x := pointAt(t, s, e, y, point.ComposeNewFlow, 0)

	_, out := s.Click(s.Hit(x, y, e), e)
	if out.Flow != New {
		t.Errorf("the New button asked for flow %q, want a flow nobody has written yet", out.Flow)
	}

	// The summary line under the row: pointing at it inspects the flow the
	// form is set to, which is the same thing clicking the chosen pill does.
	sum := s.Hit(20, y+1, e)
	if sum.Kind != point.ComposeInspectFlow {
		t.Fatalf("the summary line is kind %d, want the flow inspector", sum.Kind)
	}

	if _, out := s.Click(sum, e); out.Flow == "" || out.Flow == New {
		t.Errorf("pointing at the summary asked for flow %q, want the one it summarises", out.Flow)
	}
}

// TestTheFormsButtonsAreWhereTheyAreDrawn. Save, save and run, and cancel
// are three boxes on one line, and their widths come from the words the
// reader's language spends on them.
func TestTheFormsButtonsAreWhereTheyAreDrawn(t *testing.T) {
	s, e := form(t)

	y := formRow(t, s, e, "Save")

	var seen []string

	for x := range e.Frame.Body.W {
		at := s.Hit(x, y, e)
		if at.Kind != point.ComposeAction {
			continue
		}

		if len(seen) == 0 || seen[len(seen)-1] != at.Key {
			seen = append(seen, at.Key)
		}
	}

	want := []string{"save", "save_and_run", "cancel"}
	if len(seen) != len(want) {
		t.Fatalf("the actions row answers with %v, want %v in that order", seen, want)
	}

	for i, k := range want {
		if seen[i] != k {
			t.Fatalf("the actions row answers with %v, want %v in that order", seen, want)
		}
	}

	if _, out := s.Click(point.Target{Kind: point.ComposeAction, Key: "cancel"}, e); !out.Leave {
		t.Error("cancel left the form open")
	}
}

// TestTheOtherTabHasItsOwnRows. The form is two forms — one a task is typed
// into and one an issue is fetched into — and the second is reached by
// pointing at its tab.
func TestTheOtherTabHasItsOwnRows(t *testing.T) {
	s, e := form(t)

	tabs := formRow(t, s, e, "Manual")

	s, _ = s.Click(s.Hit(e.Frame.Body.W/2, tabs, e), e)
	if s.tab != composeTabURL {
		t.Fatalf("the right half of the tab line left tab %d, want the URL one", s.tab)
	}

	// The URL is written into a box like the task is: its label row carries
	// the paste button and answers for the field, and the lines between the
	// borders are where a reader pointing at what they pasted lands on a
	// caret.
	y := formRow(t, s, e, "url:")
	pointAt(t, s, e, y, point.ComposePaste, 0)

	for _, x := range []int{composeLabelStart, e.Frame.Body.W - 2} {
		if at := s.Hit(x, y, e); at.Kind != point.ComposeField || at.Pane != composeURL {
			t.Errorf("column %d of the url label row is kind %d pane %d, want the url field", x, at.Kind, at.Pane)
		}
	}

	if at := s.Hit(composeBoxStart, y+1, e); at.Kind != point.ComposeCaret || at.Pane != composeURL {
		t.Errorf("inside the url box is kind %d pane %d, want the caret of the url field", at.Kind, at.Pane)
	}

	// Every field of this tab is its own, not the manual tab's.
	if at := s.Hit(e.Frame.Body.W-2, formRow(t, s, e, "flow:"), e); at.Pane != composeURLFlow {
		t.Errorf("the flow row of the url tab is field %d, want the url one", at.Pane)
	}
}

// TestThePasteButtonIsNotTheTaskField. The button ends the row the label and
// the top border of the box are on, and the cell the reader lands on decides
// whether the clipboard is read or the caret moves.
func TestThePasteButtonIsNotTheTaskField(t *testing.T) {
	s, e := form(t)

	y := formRow(t, s, e, "📋 Paste")

	pointAt(t, s, e, y, point.ComposePaste, 0)

	// The rest of that row is the label and the top border, which is the
	// field itself: the box begins where the other values of the form begin.
	for _, x := range []int{cells.Gutter, composeBoxStart, e.Frame.Body.W - 2} {
		if at := s.Hit(x, y, e); at.Kind != point.ComposeField || at.Pane != composeText {
			t.Errorf("column %d of the task label row is kind %d pane %d, want the text field", x, at.Kind, at.Pane)
		}
	}

	// The row under it is the first line of the text: a reader aiming at
	// what they typed is aiming at a caret and not at the field.
	if at := s.Hit(composeBoxStart, y+1, e); at.Kind != point.ComposeCaret || at.Pane != composeText {
		t.Errorf("inside the text box is kind %d pane %d, want the caret of the task", at.Kind, at.Pane)
	}
}

// TestTheBlankRowsOfTheFormAnswerForNothing. A form that answered for every
// cell of its body would make the gap under the tabs clickable, and a reader
// who misses a row by one would move the cursor without meaning to.
func TestTheBlankRowsOfTheFormAnswerForNothing(t *testing.T) {
	s, e := form(t)

	tabs := formRow(t, s, e, "Manual")
	if at := s.Hit(2, tabs+1, e); at.Kind != point.None {
		t.Errorf("the blank line under the tabs answers with kind %d, want nothing", at.Kind)
	}
}
