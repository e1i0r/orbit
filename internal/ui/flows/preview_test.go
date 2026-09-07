package flows

// One flow read on its own: what the preview answers, and where leaving it
// goes back to.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// TestThePreviewGoesBackToWhoeverOpenedIt. The form sent the reader here to
// look at a flow, and ⏎ there is a choice; on the board it is just a way
// back to the list.
func TestThePreviewGoesBackToWhoeverOpenedIt(t *testing.T) {
	e := world(t)

	// From the form: escape leaves the designer, and ⏎ leaves it carrying
	// the flow that was being looked at.
	fromForm := Preview("careful", FromCompose, e)

	left, out := fromForm.Key(tea.KeyPressMsg{Code: tea.KeyEscape}, e)
	if !out.Leave || out.Back != FromCompose {
		t.Errorf("escape from the form's preview answered leave=%v back=%v", out.Leave, out.Back)
	}

	if left.Previewing() {
		t.Error("escape left the preview up")
	}

	_, out = fromForm.Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e)
	if !out.Leave || out.Chose != "careful" {
		t.Errorf("⏎ answered leave=%v chose=%q, want the flow it was showing", out.Leave, out.Chose)
	}

	// From the board: both keys close the preview and leave the reader on
	// the list they opened it from.
	fromBoard := Preview("careful", FromBoard, e)

	for _, msg := range []tea.KeyPressMsg{{Code: tea.KeyEscape}, {Code: tea.KeyEnter}} {
		after, out := fromBoard.Key(msg, e)
		if out.Leave || after.Previewing() {
			t.Errorf("%v on the board's preview answered leave=%v previewing=%v",
				msg, out.Leave, after.Previewing())
		}

		if !after.Listing() {
			t.Error("closing the preview did not leave the reader on the list")
		}
	}
}

// TestEOnAPreviewOpensItForEditing, which is the one gesture that turns
// reading into writing.
func TestEOnAPreviewOpensItForEditing(t *testing.T) {
	e := world(t)

	s, _ := Preview("careful", FromBoard, e).Key(press("e"), e)
	if !s.Creating() {
		t.Error("e on a preview did not open the flow for editing")
	}

	if got := s.Editing(); got != "careful" {
		t.Errorf("the designer says it is editing %q", got)
	}
}

// TestThePreviewDrawsThePhasesItWillRun.
func TestThePreviewDrawsThePhasesItWillRun(t *testing.T) {
	e := world(t)

	drawn := ansi.Strip(strings.Join(Preview("careful", FromBoard, e).View(30, 100, e), "\n"))
	if !strings.Contains(drawn, "careful") {
		t.Errorf("the preview does not name the flow:\n%s", drawn)
	}

	if !strings.Contains(strings.ToLower(drawn), "implement") {
		t.Errorf("the preview does not draw the phases:\n%s", drawn)
	}
}

// TestTheFormScrollsUnderATerminalTooShortForIt, and the wheel moves the
// list of choices instead while one is open above it.
func TestTheFormScrollsUnderATerminalTooShortForIt(t *testing.T) {
	s, e := editing(t)

	// A body of six rows is shorter than any form this screen draws.
	e.Frame.Body.H = 6

	down := s.ScrollBuilder(3, e)
	if down.scroll == 0 {
		t.Error("the form did not scroll under a window too short for it")
	}

	if up := down.ScrollBuilder(-99, e); up.scroll != 0 {
		t.Errorf("scrolling past the top left the form at %d", up.scroll)
	}

	// Past the end it stops at the last row rather than running on.
	end := s.ScrollBuilder(999, e)
	if again := end.ScrollBuilder(1, e); again.scroll != end.scroll {
		t.Errorf("scrolling past the end moved from %d to %d", end.scroll, again.scroll)
	}

	// With a list open the wheel moves the choice, not the form.
	opened := s.openPicker(flowFieldModel, e)

	turned := opened.Turn(1, e)
	if turned.picker.sel != 1 || turned.scroll != opened.scroll {
		t.Errorf("the wheel over an open list moved sel=%d scroll=%d",
			turned.picker.sel, turned.scroll)
	}

	// And with none, it scrolls the form.
	if got := s.Turn(3, e); got.scroll == 0 {
		t.Error("the wheel over the form scrolled nothing")
	}
}

// TestTheFormFollowsTheFieldTheReaderMovedTo. A window that stayed where it
// was is a reader typing into a field they cannot see.
func TestTheFormFollowsTheFieldTheReaderMovedTo(t *testing.T) {
	s, e := editing(t)
	e.Frame.Body.H = 6

	// Walk to the end of the form: the window has to come with it.
	for range 12 {
		s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyTab}, e)
	}

	lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)

	at := s.fieldRow(lines)
	if at < start || at > start+e.Frame.Body.H-1 {
		t.Errorf("the field is on row %d and the window shows %d to %d",
			at, start, start+e.Frame.Body.H-1)
	}

	// A window with room for the whole form never scrolls.
	tall := s
	e.Frame.Body.H = 200

	if got := tall.followField(e); got.scroll != 0 {
		t.Errorf("a window with room for the form scrolled to %d", got.scroll)
	}
}

// TestTheTabStripIsReachedWithControlAndAnArrow, which no field on any tab
// uses.
func TestTheTabStripIsReachedWithControlAndAnArrow(t *testing.T) {
	s, e := editing(t)

	was := s.tab

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl}, e)
	if s.tab == was {
		t.Error("ctrl+→ did not move to the next tab")
	}

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl}, e)
	if s.tab != was {
		t.Errorf("ctrl+← left the reader on tab %d, want %d", s.tab, was)
	}

	// Without the modifier the arrows belong to the field under them.
	if _, held := flowTabKey(tea.KeyPressMsg{Code: tea.KeyRight}); held {
		t.Error("a bare arrow was taken by the tab strip")
	}

	if _, held := flowTabKey(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl}); held {
		t.Error("ctrl and a letter was taken by the tab strip")
	}
}
