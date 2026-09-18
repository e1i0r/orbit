package flows

// The clicks the designer answers that nothing was pressing.
//
// template_click_test.go has the buttons; these are the rest of what Hit can
// name — the preview, the two ways out of it, the tab strip, the engine and
// model pickers, and taking an option out of one. A click the screen does
// not answer is a control a reader can see and cannot use, which is worse
// than one that is not drawn.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/point"
)

// TestClickingAFlowOpensItsPreview. The list says a flow's name and its
// phase count; what it does is in the preview, and a reader who clicked the
// row meant to read it.
func TestClickingAFlowOpensItsPreview(t *testing.T) {
	s, e := designer(t)

	name := firstListed(t, s)

	next, _ := s.Click(point.Target{Field: "details", ID: name}, e)

	if !next.showingDetail {
		t.Error("clicking a flow did not open what it does")
	}
}

// TestBothWaysOutOfThePreviewAnswerAClick. One of them takes the flow and
// the other leaves it, and on the board they both go back to the list — a
// preview with no way out but a key is a reader stuck in a panel.
func TestBothWaysOutOfThePreviewAnswerAClick(t *testing.T) {
	for _, way := range []string{"detail_select", "detail_back"} {
		s, e := designer(t)

		name := firstListed(t, s)

		open, _ := s.Click(point.Target{Field: "details", ID: name}, e)

		shut, _ := open.Click(point.Target{Field: way}, e)
		if shut.showingDetail {
			t.Errorf("clicking %q left the preview open", way)
		}
	}
}

// TestChoosingAFlowFromAComposeLeavesWithIt. The designer opened from the
// compose screen is a picker: what the reader came for is the name, and the
// screen hands it back rather than returning to a list they did not ask for.
func TestChoosingAFlowFromAComposeLeavesWithIt(t *testing.T) {
	e := world(t)
	s := Open(FromCompose, e)

	name := firstListed(t, s)

	open, _ := s.Click(point.Target{Field: "details", ID: name}, e)

	next, out := open.Click(point.Target{Field: "detail_select"}, e)

	if !out.Leave {
		t.Error("choosing a flow from a compose did not leave the screen")
	}

	if out.Chose == "" {
		t.Error("it left without saying which flow was chosen")
	}

	if next.showingDetail {
		t.Error("it left with the preview still open")
	}

	// And backing out of the preview leaves too, with nothing chosen: a
	// reader who opened the picker and changed their mind is not made to
	// pick something.
	away, out := open.Click(point.Target{Field: "detail_back"}, e)
	if !out.Leave {
		t.Error("backing out of a compose picker did not leave the screen")
	}

	if away.showingDetail {
		t.Error("it left with the preview still open")
	}
}

// TestTheTabStripAnswersAClick. The builder has two tabs and one of them is
// the only way to write a flow from a sentence; a strip drawn and not
// clickable is the reader's hand going to the keyboard for a thing that
// looks like a button.
func TestTheTabStripAnswersAClick(t *testing.T) {
	s, e := editing(t)

	other := flowTabSay
	if s.tab == flowTabSay {
		other = flowTabFields
	}

	next, _ := s.Click(point.Target{Field: "tab", Phase: other}, e)

	if next.tab != other {
		t.Errorf("clicking the tab strip left the screen on tab %v, want %v", next.tab, other)
	}

	// And the new tab starts at its top rather than wherever the last one
	// was scrolled to.
	if next.scroll != 0 {
		t.Errorf("the tab opened scrolled to %d", next.scroll)
	}
}

// TestTheEngineAndModelDialsOpenUnderAClick. They are lists of what this
// build actually has, and a dial that only the keyboard opens is a dial most
// readers never find.
func TestTheEngineAndModelDialsOpenUnderAClick(t *testing.T) {
	for _, dial := range []string{"say_engine", "say_model"} {
		s, e := editing(t)

		next, _ := s.Click(point.Target{Field: dial}, e)

		if !next.picker.open {
			t.Errorf("clicking %q opened no list", dial)
		}
	}
}

// TestAnOptionTakenWithThePointerIsTheOneItLandedOn, which is the same
// gesture enter is on the row the cursor is already on.
func TestAnOptionTakenWithThePointerIsTheOneItLandedOn(t *testing.T) {
	s, e := editing(t)

	open, _ := s.Click(point.Target{Field: "say_engine"}, e)
	if !open.picker.open {
		t.Fatal("the engine list did not open")
	}

	next, _ := open.Click(point.Target{Field: "pick", Phase: 0}, e)

	if next.picker.open {
		t.Error("taking an option with the pointer left the list open")
	}
}

// TestAClickWithNoNameOnItIsAClickOnTheFieldItLandedOn.
//
// The rows of the builder carry no name of their own — a field is known by
// its number — so the default of the switch is not a hole: it is how every
// row of the form is clicked. What it must not be is a click that does
// whatever the last case happened to do, and the test for that is that the
// cursor ends up on the field the pointer named.
func TestAClickWithNoNameOnItIsAClickOnTheFieldItLandedOn(t *testing.T) {
	s, e := editing(t)

	next, _ := s.Click(point.Target{Phase: flowFieldEngine}, e)

	if next.field != flowFieldEngine {
		t.Errorf("the cursor is on field %d, want the one the pointer named", next.field)
	}

	if !next.picker.open {
		t.Error("clicking a dial with a list behind it opened no list")
	}
}

// firstListed is the name of the first flow the designer is showing, which
// on a machine with no flows of its own is one of the built-in ones.
func firstListed(t *testing.T, s State) string {
	t.Helper()

	if len(s.listed) == 0 {
		t.Fatal("the designer is listing no flows at all")
	}

	name := s.listed[0].Name
	if strings.TrimSpace(name) == "" {
		t.Fatal("the first flow listed has no name")
	}

	return name
}
