package prose

// The grid: every cell the same width, every choice drawn.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func flows(n int) []Choice {
	names := []string{"careful", "cobertura", "coverage", "gated", "quick", "safe-refactor", "task", "tdd-cycle"}

	out := make([]Choice, 0, n)
	for _, name := range names[:n] {
		out = append(out, Choice{Label: name, Glyph: FlowGlyph(name)})
	}

	return out
}

// TestEveryCellIsTheSameWidth is what makes it a table rather than a line
// of pills: laid out as they came, the second row began wherever the first
// happened to end and the words under each other were half a pill apart.
func TestEveryCellIsTheSameWidth(t *testing.T) {
	_, placed := Row(flows(8), 0, 12, 100)

	if len(placed) != 8 {
		t.Fatalf("eight choices were placed as %d", len(placed))
	}

	wide := placed[0].Cells
	for i, p := range placed {
		if p.Cells != wide {
			t.Errorf("cell %d is %d wide and cell 0 is %d", i, p.Cells, wide)
		}
	}

	// And the columns line up: the same column on two lines is the same
	// number of cells from the left.
	byColumn := map[int][]int{}
	for _, p := range placed {
		byColumn[p.Line] = append(byColumn[p.Line], p.At)
	}

	first := byColumn[0]
	for line, ats := range byColumn {
		for col, at := range ats {
			if col < len(first) && at != first[col] {
				t.Errorf("line %d column %d starts at %d, and line 0 starts it at %d",
					line, col, at, first[col])
			}
		}
	}
}

// TestEveryChoiceIsDrawn. A grid that cuts the last three off at a hundred
// columns is the failure this exists to end, with a tidier first line.
func TestEveryChoiceIsDrawn(t *testing.T) {
	for _, w := range []int{60, 100, 200} {
		choices := flows(8)

		lines, placed := Row(choices, 0, 12, w)
		drawn := strings.Join(lines, "\n")

		for _, c := range choices {
			if !strings.Contains(drawn, c.Label) {
				t.Errorf("at %d columns %q is not drawn:\n%s", w, c.Label, drawn)
			}
		}

		if len(placed) != len(choices) {
			t.Errorf("at %d columns %d of %d choices were placed", w, len(placed), len(choices))
		}

		for i, l := range lines {
			if got := lipgloss.Width(l); w > 0 && got > w {
				t.Errorf("at %d columns line %d is %d wide", w, i, got)
			}
		}
	}
}

// TestAWidthOfNoughtDoesNotWrap: the compose form places the box under
// this row from a fixed plan, so a second line would move everything.
func TestAWidthOfNoughtDoesNotWrap(t *testing.T) {
	lines, _ := Row(flows(8), 0, 0, 0)
	if len(lines) != 1 {
		t.Errorf("a width of nought wrapped onto %d lines", len(lines))
	}
}

// TestTheMarkerKeepsItsRoomOnEveryCell, or a name shifts two columns left
// the moment it stops being the one in use.
func TestTheMarkerKeepsItsRoomOnEveryCell(t *testing.T) {
	first, _ := Row(flows(3), 0, 0, 0)
	last, _ := Row(flows(3), 2, 0, 0)

	if lipgloss.Width(first[0]) != lipgloss.Width(last[0]) {
		t.Errorf("the row is %d wide with the first marked and %d with the last",
			lipgloss.Width(first[0]), lipgloss.Width(last[0]))
	}
}

// TestAtAnswersByLineAndColumn: the same column two lines down is a
// different choice.
func TestAtAnswersByLineAndColumn(t *testing.T) {
	_, placed := Row(flows(8), 0, 12, 100)

	for want, p := range placed {
		got, on := At(placed, p.At, p.Line)
		if !on || got != want {
			t.Errorf("the cell placed at (%d, line %d) answers (%d, %v)", p.At, p.Line, got, on)
		}
	}

	if _, on := At(placed, 0, 0); on {
		t.Error("column 0 is inside the label and answered a choice")
	}
}
