package ui

// The form's fields seen from the keyboard and the pointer, which is where
// the complaint came from: what was typed did not go where the reader was
// looking, and clicking inside the box did nothing at all.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// composeOn is the form with the cursor on one field and something already
// written in it.
func composeOn(t *testing.T, field int, val string) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m = m.openCompose()
	m.compose.field = field

	in := m.compose.active()
	if in == nil {
		t.Fatalf("field %d is not one that is typed into", field)
	}

	in.setValue(val)

	return m
}

func TestWhatIsTypedGoesWhereTheCaretIs(t *testing.T) {
	m := composeOn(t, composeID, "ORBIT-42")
	m.compose.id.moveTo(5)

	m = asModel(t, mustUpdate(m, press("x")))

	if got := m.compose.id.String(); got != "ORBITx-42" {
		t.Errorf("typing in the middle of the id left %q", got)
	}
}

// The arrows are the caret in a field that is typed into and the pills on a
// row of pills. One key, and which it is depends on where the reader is.
func TestTheArrowsMoveTheCaretAndStillCycleThePills(t *testing.T) {
	m := composeOn(t, composeText, "hola")

	m = asModel(t, mustUpdate(m, tea.KeyPressMsg{Code: tea.KeyLeft}))
	if m.compose.text.at != 3 {
		t.Errorf("the left arrow left the caret at %d, want 3", m.compose.text.at)
	}

	m = asModel(t, mustUpdate(m, tea.KeyPressMsg{Code: tea.KeyRight}))
	if m.compose.text.at != 4 {
		t.Errorf("the right arrow left the caret at %d, want 4", m.compose.text.at)
	}

	m.compose.field = composeFlow
	was := m.compose.flowIdx

	m = asModel(t, mustUpdate(m, tea.KeyPressMsg{Code: tea.KeyRight}))
	if len(m.compose.flows) > 1 && m.compose.flowIdx == was {
		t.Error("the right arrow on the flow row did not cycle the pills")
	}
}

// Up and down are the lines of the task while there are lines left, and the
// fields of the form once there are not.
func TestUpAndDownWalkTheLinesOfTheTaskAndThenLeaveTheField(t *testing.T) {
	m := composeOn(t, composeText, "uno\ndos")
	m.compose.text.moveTo(0)

	m = asModel(t, mustUpdate(m, press("down")))
	if m.compose.field != composeText || m.compose.text.at != 4 {
		t.Errorf("down inside the box left field %d caret %d, want the second line", m.compose.field, m.compose.text.at)
	}

	m = asModel(t, mustUpdate(m, press("up")))
	if m.compose.field != composeText || m.compose.text.at != 0 {
		t.Errorf("up inside the box left field %d caret %d, want the first line", m.compose.field, m.compose.text.at)
	}

	// There is no line above the first one, so the arrow means what it
	// means on every other row of the form: the field above.
	m = asModel(t, mustUpdate(m, press("up")))
	if m.compose.field != composeID {
		t.Errorf("up on the first line left field %d, want the id above the box", m.compose.field)
	}
}

// The one the complaint was about: pointing inside the box puts the caret
// where it was pointed, and what is typed next goes there.
func TestPointingInsideTheBoxPutsTheCaretThere(t *testing.T) {
	m := composeOn(t, composeText, "hola mundo")
	y := formRow(t, m, "hola mundo")

	at := m.hit(composeBoxStart+4, y)
	if at.Kind != TargetComposeCaret || at.Pane != composeText {
		t.Fatalf("a cell inside the box is kind %d pane %d, want the caret of the task", at.Kind, at.Pane)
	}

	after, _ := m.leftClick(at)

	m = asModel(t, after)
	if m.compose.text.at != 4 {
		t.Fatalf("the click left the caret at %d, want 4", m.compose.text.at)
	}

	m = asModel(t, mustUpdate(m, press("!")))
	if got := m.compose.text.String(); got != "hola! mundo" {
		t.Errorf("typing after the click left %q", got)
	}
}

// A one-line field answers the same way, counting from where its value
// starts rather than from the edge of the screen.
func TestPointingAtAOneLineFieldPutsTheCaretThere(t *testing.T) {
	m := composeOn(t, composeID, "ORBIT-42")
	y := formRow(t, m, "ORBIT-42")

	after, _ := m.leftClick(m.hit(composeLabelStart+3, y))

	m = asModel(t, after)
	if m.compose.id.at != 3 {
		t.Errorf("the click left the caret at %d, want 3", m.compose.id.at)
	}
}

// The block is drawn on the caret. It used to be hung off the last line the
// box drew, which with one line written and the box padded out to three was
// two rows below the text.
func TestTheBlockIsDrawnOnTheCaretAndNotUnderTheText(t *testing.T) {
	m := composeOn(t, composeText, "adaa")
	m.compose.text.moveTo(2)

	lines := m.composeBoxLines(m.compose.text, 40, true, "")
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
