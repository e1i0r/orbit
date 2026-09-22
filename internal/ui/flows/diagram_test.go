package flows

// The pipeline diagram: how wide a box is, where the boxes sit, and what
// happens when the window has no room for them side by side.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/flow"
)

// phasesNamed is a pipeline of phases with the given names.
func phasesNamed(names ...string) []flow.Phase {
	out := make([]flow.Phase, 0, len(names))
	for _, n := range names {
		out = append(out, flow.Phase{Name: n, Engine: "claude", Model: "opus"})
	}

	return out
}

// TestNoBoxOfTheDiagramRunsPastTheRoomItWasGiven. The stacked form the
// diagram falls back to was never measured against anything, so a phase
// with a long name drew a box past the right of the window — which the
// terminal wraps, putting every row of the reading below it a row down
// from where the reader is looking.
func TestNoBoxOfTheDiagramRunsPastTheRoomItWasGiven(t *testing.T) {
	for _, phases := range [][]flow.Phase{
		phasesNamed("implement"),
		phasesNamed("implement", "review", "fix"),
		phasesNamed("implement-the-whole-payments-reconciliation", "review"),
		phasesNamed(strings.Repeat("x", 120)),
	} {
		for maxW := 10; maxW <= 160; maxW++ {
			for i, r := range renderFlowDiagram(phases, maxW) {
				if got := lipgloss.Width(r); got > maxW {
					t.Fatalf("with %d phases in %d columns, row %d is %d cells wide: %q",
						len(phases), maxW, i, got, ansi.Strip(r))
				}
			}
		}
	}
}

// TestABoxIsAsWideAsWhatIsInIt, and the four rows of it are one box: a top
// and a bottom that do not agree with the sides between them is not a box,
// it is four lines that happen to be near each other.
func TestABoxIsAsWideAsWhatIsInIt(t *testing.T) {
	for _, c := range []struct {
		name string
		want int
	}{
		// The floor: a one-word phase is not a tick box.
		{"fix", diagramBoxFloor},
		{"implement", diagramBoxFloor},
		// Past it, the name and the two sides.
		{
			"implement-the-payments-side",
			lipgloss.Width("1. implement-the-payments-side") + 2,
		},
	} {
		rows := renderFlowDiagram(phasesNamed(c.name), 200)
		if len(rows) != 4 {
			t.Fatalf("one phase drew %d rows, want the four of one box", len(rows))
		}

		for i, r := range rows {
			if got := lipgloss.Width(r); got != c.want {
				t.Errorf("%q: row %d of the box is %d cells wide, want %d", c.name, i, got, c.want)
			}
		}
	}
}

// TestTheBoxesSitSideBySideWhileThereIsRoom, and stack when there is not.
func TestTheBoxesSitSideBySideWhileThereIsRoom(t *testing.T) {
	phases := phasesNamed("implement", "review", "fix")

	wide := renderFlowDiagram(phases, 200)
	if len(wide) != 4 {
		t.Fatalf("three phases in two hundred columns drew %d rows, want four", len(wide))
	}

	joined := ansi.Strip(strings.Join(wide, "\n"))
	for _, name := range []string{"implement", "review", "fix"} {
		if !strings.Contains(joined, name) {
			t.Errorf("the row of boxes does not say %q:\n%s", name, joined)
		}
	}

	if !strings.Contains(ansi.Strip(wide[1]), "▶") {
		t.Errorf("the boxes are not joined by anything: %q", ansi.Strip(wide[1]))
	}

	// Four rows a box, and a connector between each pair.
	narrow := renderFlowDiagram(phases, 30)
	if len(narrow) != 4*len(phases)+2*(len(phases)-1) {
		t.Fatalf("three phases in thirty columns drew %d rows, want a stack", len(narrow))
	}

	stacked := ansi.Strip(strings.Join(narrow, "\n"))
	if !strings.Contains(stacked, "▼") {
		t.Errorf("the stacked boxes are not joined by anything:\n%s", stacked)
	}

	for _, name := range []string{"implement", "review", "fix"} {
		if !strings.Contains(stacked, name) {
			t.Errorf("the stack does not say %q:\n%s", name, stacked)
		}
	}
}

// TestTheFourRowsOfABoxLineUp. The corners of the top have to be the
// corners of the bottom, or the sides between them belong to nothing.
func TestTheFourRowsOfABoxLineUp(t *testing.T) {
	rows := renderFlowDiagram(phasesNamed("implement", "review"), 200)
	if len(rows) != 4 {
		t.Fatalf("two phases drew %d rows", len(rows))
	}

	corners := func(row string, open, close rune) []int {
		var at []int

		for i, r := range []rune(ansi.Strip(row)) {
			if r == open || r == close {
				at = append(at, i)
			}
		}

		return at
	}

	top := corners(rows[0], '┌', '┐')
	bot := corners(rows[3], '└', '┘')

	if len(top) != len(bot) {
		t.Fatalf("the top has %d corners and the bottom %d", len(top), len(bot))
	}

	for i := range top {
		if top[i] != bot[i] {
			t.Errorf("corner %d is at column %d on the top and %d on the bottom", i, top[i], bot[i])
		}
	}

	// And the sides are where the corners are.
	for _, row := range rows[1:3] {
		sides := corners(row, '│', '│')
		if len(sides) != len(top) {
			t.Fatalf("a side row has %d walls and the box has %d corners", len(sides), len(top))
		}

		for i := range sides {
			if sides[i] != top[i] {
				t.Errorf("wall %d is at column %d and its corner at %d", i, sides[i], top[i])
			}
		}
	}
}

// TestAPipelineOfNothingDrawsNothing, which is a flow being written before
// its first phase is added.
func TestAPipelineOfNothingDrawsNothing(t *testing.T) {
	if got := renderFlowDiagram(nil, 80); got != nil {
		t.Errorf("no phases drew %d rows", len(got))
	}
}

// TestALoopSaysHowManyTurnsItTakes, where a plain phase says what runs it.
func TestALoopSaysHowManyTurnsItTakes(t *testing.T) {
	phases := []flow.Phase{
		{Name: "tests", Engine: "claude", Model: "opus", Loop: &flow.Loop{Max: 4}},
		{Name: "review", Engine: "claude", Model: "opus", Wait: true},
	}

	drawn := ansi.Strip(strings.Join(renderFlowDiagram(phases, 200), "\n"))

	for _, want := range []string{"↻ ×4", "⏸", "claude/opus"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the diagram does not say %q:\n%s", want, drawn)
		}
	}
}
