package compose

// Selecting text in the form, by keyboard and by pointer. What was reported
// is one sentence — the text cannot be selected — and it is three things:
// shift held with a movement, a drag with the button down, and something
// drawn to show what was taken.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/ui/typing"
)

func TestShiftHeldWithAnArrowSelectsWhatItCrosses(t *testing.T) {
	s, e := composeOn(t, composeID, "ORBIT-42")
	s.id.MoveTo(0)

	for range 5 {
		s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}, e)
	}

	if got := s.id.Selected(); got != "ORBIT" {
		t.Errorf("five shifted arrows took %q, want %q", got, "ORBIT")
	}

	// And what is typed next replaces it, which is the whole point of
	// having taken it.
	s, _ = s.Key(press("x"), e)

	if got := s.id.String(); got != "x-42" {
		t.Errorf("typing over the selection left %q", got)
	}
}

// Shift with up and down is the lines of the box, the same movement the
// bare arrows make.
func TestShiftHeldWithDownSelectsToTheLineBelow(t *testing.T) {
	s, e := composeOn(t, composeText, "uno\ndos")
	s.text.MoveTo(0)

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}, e)

	if got := s.text.Selected(); got != "uno\n" {
		t.Errorf("shift with down took %q, want the first line", got)
	}
}

// A movement that leaves the field selects nothing: there is no stretch of
// text between the box and the field above it.
func TestAShiftedArrowThatLeavesTheFieldTakesNothingWithIt(t *testing.T) {
	s, e := composeOn(t, composeText, "uno")
	s.text.MoveTo(0)

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}, e)

	if s.field != composeID {
		t.Fatalf("up on the first line left field %d, want the id above the box", s.field)
	}

	if s.text.HasSelection() {
		t.Errorf("leaving the box left %q selected", s.text.Selected())
	}
}

// Selected text is drawn as selected, and nothing about the value changes:
// what is painted over is still every character that was there.
func TestWhatIsSelectedIsDrawnAsSelected(t *testing.T) {
	s, _ := composeOn(t, composeText, "hola mundo")
	s.text.MoveTo(0)
	s.text.Extend(func(in *typing.Field) { in.MoveTo(4) })

	lines := s.composeBoxLines(s.text, 40, true, "")
	if len(lines) == 0 {
		t.Fatal("the box drew nothing")
	}

	if !strings.Contains(lines[0], theme.Paint(theme.Sel).Render("hola")) {
		t.Errorf("the selected stretch is not painted as selected: %q", lines[0])
	}

	if got := ansi.Strip(lines[0]); got != "hola mundo" {
		t.Errorf("the line reads %q once the paint is taken off, want the value", got)
	}
}
