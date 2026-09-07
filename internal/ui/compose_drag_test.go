package ui

// A selection dragged out of the form with the pointer. The drag is the
// window's: press, move and release are routed by its own mouse loop, and
// what they leave behind is read off the form the way a reader reads it —
// the stretch that is painted as selected.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// writing is the form with the cursor in the box a task goes in and
// something already written there, typed the way a reader types it.
func writing(t *testing.T, val string) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m = m.openCompose()

	m = asModel(t, mustUpdate(m, tea.KeyPressMsg{Code: tea.KeyDown}))
	m = asModel(t, mustUpdate(m, tea.KeyPressMsg{Code: tea.KeyDown}))

	for _, r := range val {
		m = asModel(t, mustUpdate(m, tea.KeyPressMsg{Code: r, Text: string(r)}))
	}

	return m
}

// selectedOnScreen is the stretch the form is painting as selected, which is
// the only thing a reader can see of a selection.
func selectedOnScreen(m Model, want string) bool {
	drawn := strings.Join(m.composeRows(m.frame.Body.H, m.frame.Body.W), "\n")

	return strings.Contains(drawn, theme.Paint(theme.Sel).Render(want))
}

// caretCell is the column the value on that row starts in: the first cell
// that points at the head of the text and whose neighbour points at the
// character after it. Everything left of it clamps to the head, so a drag
// that started there would never cross anything.
func caretCell(t *testing.T, m Model, y int) int {
	t.Helper()

	for x := range m.width - 1 {
		at, next := m.hit(x, y), m.hit(x+1, y)
		if at.Kind == point.ComposeCaret && at.Caret == 0 && next.Caret == 1 {
			return x
		}
	}

	t.Fatalf("no cell of row %d is the head of a value", y)

	return -1
}

// TestDraggingThePointerSelectsWhatItCrossed. The press has to act rather
// than the release, because by the time the button comes up the pointer is
// somewhere else.
func TestDraggingThePointerSelectsWhatItCrossed(t *testing.T) {
	m := writing(t, "hola mundo")

	y := rowOf(screenRows(m), "hola mundo")
	if y < 0 {
		t.Fatal("what was typed is not on the screen")
	}

	from := caretCell(t, m, y)

	held := pointed(t, m, tea.MouseClickMsg{X: from, Y: y, Button: tea.MouseLeft})
	dragged := pointed(t, held, tea.MouseMotionMsg{X: from + 4, Y: y, Button: tea.MouseLeft})

	if !selectedOnScreen(dragged, "hola") {
		t.Error("the drag selected nothing")
	}

	// Letting go where the drag ended keeps it: the release lands on a
	// different cell from the press, so it is not a click on anything.
	after := pointed(t, dragged, tea.MouseReleaseMsg{X: from + 4, Y: y, Button: tea.MouseLeft})
	if !selectedOnScreen(after, "hola") {
		t.Error("letting go dropped the selection the drag had made")
	}
}

// TestAClickThatDidNotMoveSelectsNothing.
func TestAClickThatDidNotMoveSelectsNothing(t *testing.T) {
	m := writing(t, "hola mundo")

	y := rowOf(screenRows(m), "hola mundo")
	if y < 0 {
		t.Fatal("what was typed is not on the screen")
	}

	at := caretCell(t, m, y) + 4

	held := pointed(t, m, tea.MouseClickMsg{X: at, Y: y, Button: tea.MouseLeft})

	after := pointed(t, held, tea.MouseReleaseMsg{X: at, Y: y, Button: tea.MouseLeft})
	if selectedOnScreen(after, "hola") {
		t.Error("a plain click left a selection behind it")
	}
}

// mustUpdate is one message through the window's own loop.
func mustUpdate(m Model, msg tea.Msg) tea.Model {
	next, _ := m.Update(msg)

	return next
}
