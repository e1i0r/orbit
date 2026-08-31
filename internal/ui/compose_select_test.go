package ui

// Selecting text in the form, by keyboard and by pointer. What was reported
// is one sentence — the text cannot be selected — and it is three things:
// shift held with a movement, a drag with the button down, and something
// drawn to show what was taken.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestShiftHeldWithAnArrowSelectsWhatItCrosses(t *testing.T) {
	m := composeOn(t, composeID, "ORBIT-42")
	m.compose.id.moveTo(0)

	for range 5 {
		m = asModel(t, mustUpdate(m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}))
	}

	if got := m.compose.id.selected(); got != "ORBIT" {
		t.Errorf("five shifted arrows took %q, want %q", got, "ORBIT")
	}

	// And what is typed next replaces it, which is the whole point of
	// having taken it.
	m = asModel(t, mustUpdate(m, press("x")))

	if got := m.compose.id.String(); got != "x-42" {
		t.Errorf("typing over the selection left %q", got)
	}
}

// Shift with up and down is the lines of the box, the same movement the
// bare arrows make.
func TestShiftHeldWithDownSelectsToTheLineBelow(t *testing.T) {
	m := composeOn(t, composeText, "uno\ndos")
	m.compose.text.moveTo(0)

	m = asModel(t, mustUpdate(m, tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}))

	if got := m.compose.text.selected(); got != "uno\n" {
		t.Errorf("shift with down took %q, want the first line", got)
	}
}

// A movement that leaves the field selects nothing: there is no stretch of
// text between the box and the field above it.
func TestAShiftedArrowThatLeavesTheFieldTakesNothingWithIt(t *testing.T) {
	m := composeOn(t, composeText, "uno")
	m.compose.text.moveTo(0)

	m = asModel(t, mustUpdate(m, tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}))

	if m.compose.field != composeID {
		t.Fatalf("up on the first line left field %d, want the id above the box", m.compose.field)
	}

	if m.compose.text.hasSelection() {
		t.Errorf("leaving the box left %q selected", m.compose.text.selected())
	}
}

// The pointer's own way of saying the same thing: press on one cell, move
// with the button down, and what was crossed is selected. The press has to
// act rather than the release, because by the time the button comes up the
// pointer is somewhere else.
func TestDraggingThePointerSelectsWhatItCrossed(t *testing.T) {
	m := composeOn(t, composeText, "hola mundo")
	y := formRow(t, m, "hola mundo")

	held := pointed(t, m, tea.MouseClickMsg{X: composeBoxStart, Y: y, Button: tea.MouseLeft})
	if held.compose.text.at != 0 {
		t.Fatalf("the press left the caret at %d, want the head of the value", held.compose.text.at)
	}

	dragged := pointed(t, held, tea.MouseMotionMsg{X: composeBoxStart + 4, Y: y, Button: tea.MouseLeft})

	if got := dragged.compose.text.selected(); got != "hola" {
		t.Errorf("the drag took %q, want %q", got, "hola")
	}

	// Letting go where the drag ended keeps it: the release lands on a
	// different cell from the press, so it is not a click on anything.
	after := pointed(t, dragged, tea.MouseReleaseMsg{X: composeBoxStart + 4, Y: y, Button: tea.MouseLeft})
	if got := after.compose.text.selected(); got != "hola" {
		t.Errorf("letting go left %q selected, want the drag to stand", got)
	}
}

// A click that never moved is a caret and not a selection.
func TestAClickThatDidNotMoveSelectsNothing(t *testing.T) {
	m := composeOn(t, composeText, "hola mundo")
	y := formRow(t, m, "hola mundo")

	held := pointed(t, m, tea.MouseClickMsg{X: composeBoxStart + 4, Y: y, Button: tea.MouseLeft})
	after := pointed(t, held, tea.MouseReleaseMsg{X: composeBoxStart + 4, Y: y, Button: tea.MouseLeft})

	if after.compose.text.hasSelection() {
		t.Errorf("a plain click left %q selected", after.compose.text.selected())
	}

	if after.compose.text.at != 4 {
		t.Errorf("the click left the caret at %d, want 4", after.compose.text.at)
	}
}

// Selected text is drawn as selected, and nothing about the value changes:
// what is painted over is still every character that was there.
func TestWhatIsSelectedIsDrawnAsSelected(t *testing.T) {
	m := composeOn(t, composeText, "hola mundo")
	m.compose.text.moveTo(0)
	m.compose.text.extend(func(in *input) { in.moveTo(4) })

	lines := m.composeTextLines(40, true, "")
	if len(lines) == 0 {
		t.Fatal("the box drew nothing")
	}

	if !strings.Contains(lines[0], Paint(Sel).Render("hola")) {
		t.Errorf("the selected stretch is not painted as selected: %q", lines[0])
	}

	if got := ansi.Strip(lines[0]); got != "hola mundo" {
		t.Errorf("the line reads %q once the paint is taken off, want the value", got)
	}
}
