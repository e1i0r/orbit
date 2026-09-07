package compose

// The form's fields seen from the keyboard and the pointer, which is where
// the complaint came from: what was typed did not go where the reader was
// looking, and clicking inside the box did nothing at all.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/point"
)

// composeOn is the form with the cursor on one field and something already
// written in it.
func composeOn(t *testing.T, field int, val string) (State, Env) {
	t.Helper()

	s, e := form(t)
	s.field = field

	in := s.active()
	if in == nil {
		t.Fatalf("field %d is not one that is typed into", field)
	}

	in.SetValue(val)

	return s, e
}

func TestWhatIsTypedGoesWhereTheCaretIs(t *testing.T) {
	s, e := composeOn(t, composeID, "ORBIT-42")
	s.id.MoveTo(5)

	s, _ = s.Key(press("x"), e)

	if got := s.id.String(); got != "ORBITx-42" {
		t.Errorf("typing in the middle of the id left %q", got)
	}
}

// The arrows are the caret in a field that is typed into and the pills on a
// row of pills. One key, and which it is depends on where the reader is.
func TestTheArrowsMoveTheCaretAndStillCycleThePills(t *testing.T) {
	s, e := composeOn(t, composeText, "hola")

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyLeft}, e)
	if s.text.At != 3 {
		t.Errorf("the left arrow left the caret at %d, want 3", s.text.At)
	}

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyRight}, e)
	if s.text.At != 4 {
		t.Errorf("the right arrow left the caret at %d, want 4", s.text.At)
	}

	s.field = composeFlow
	was := s.flowIdx

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyRight}, e)
	if len(s.flows) > 1 && s.flowIdx == was {
		t.Error("the right arrow on the flow row did not cycle the pills")
	}
}

// Up and down are the lines of the task while there are lines left, and the
// fields of the form once there are not.
func TestUpAndDownWalkTheLinesOfTheTaskAndThenLeaveTheField(t *testing.T) {
	s, e := composeOn(t, composeText, "uno\ndos")
	s.text.MoveTo(0)

	s, _ = s.Key(press("down"), e)
	if s.field != composeText || s.text.At != 4 {
		t.Errorf("down inside the box left field %d caret %d, want the second line", s.field, s.text.At)
	}

	s, _ = s.Key(press("up"), e)
	if s.field != composeText || s.text.At != 0 {
		t.Errorf("up inside the box left field %d caret %d, want the first line", s.field, s.text.At)
	}

	// There is no line above the first one, so the arrow means what it
	// means on every other row of the form: the field above.
	s, _ = s.Key(press("up"), e)
	if s.field != composeID {
		t.Errorf("up on the first line left field %d, want the id above the box", s.field)
	}
}

// The one the complaint was about: pointing inside the box puts the caret
// where it was pointed, and what is typed next goes there.
func TestPointingInsideTheBoxPutsTheCaretThere(t *testing.T) {
	s, e := composeOn(t, composeText, "hola mundo")
	y := formRow(t, s, e, "hola mundo")

	at := s.Hit(composeBoxStart+4, y, e)
	if at.Kind != point.ComposeCaret || at.Pane != composeText {
		t.Fatalf("a cell inside the box is kind %d pane %d, want the caret of the task", at.Kind, at.Pane)
	}

	s, _ = s.Click(at, e)
	if s.text.At != 4 {
		t.Fatalf("the click left the caret at %d, want 4", s.text.At)
	}

	s, _ = s.Key(press("!"), e)
	if got := s.text.String(); got != "hola! mundo" {
		t.Errorf("typing after the click left %q", got)
	}
}

// A one-line field answers the same way, counting from where its value
// starts rather than from the edge of the screen.
func TestPointingAtAOneLineFieldPutsTheCaretThere(t *testing.T) {
	s, e := composeOn(t, composeID, "ORBIT-42")
	y := formRow(t, s, e, "ORBIT-42")

	s, _ = s.Click(s.Hit(composeLabelStart+3, y, e), e)
	if s.id.At != 3 {
		t.Errorf("the click left the caret at %d, want 3", s.id.At)
	}
}

// The block is drawn on the caret. It used to be hung off the last line the
// box drew, which with one line written and the box padded out to three was
// two rows below the text.
func TestTheBlockIsDrawnOnTheCaretAndNotUnderTheText(t *testing.T) {
	s, _ := composeOn(t, composeText, "adaa")
	s.text.MoveTo(2)

	lines := s.composeBoxLines(s.text, 40, true, "")
	if len(lines) < 3 {
		t.Fatalf("the box drew %d lines, want at least three", len(lines))
	}

	if !hasCaret(lines[0]) {
		t.Errorf("the first line is %q, and the caret is not on it", lines[0])
	}

	for i, l := range lines[1:] {
		if hasCaret(l) {
			t.Errorf("line %d is %q, and the caret is on it as well", i+1, l)
		}
	}
}

// hasCaret says whether a drawn line carries the block. Inside the box
// nothing else is painted, so an escape on the line is the caret.
func hasCaret(line string) bool {
	return strings.Contains(line, "\x1b[")
}
