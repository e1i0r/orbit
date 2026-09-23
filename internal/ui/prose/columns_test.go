package prose

// What keeps one column of a Fields row out of the next.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestAFullCellDoesNotTouchTheNext.
//
// Padding was the only separation between columns, so a value that
// exactly filled its cell got none of it and the next column began
// against it. At a hundred columns the cell is 25 and the engine dial
// reads "agy gemini-3.8-flash-high", which is 25: the screen showed
// "agy gemini-3.8-flash-highhigh" with the effort column apparently
// empty, and the number it broke at was the width Elio works on.
func TestAFullCellDoesNotTouchTheNext(t *testing.T) {
	const width, columns = 100, 4

	rows := Fields([]Field{
		{Label: "engine", Key: "k", Value: "agy gemini-3.8-flash-high"},
		{Label: "effort", Key: "E", Value: "high"},
		{Label: "thinking", Key: "t", Value: "adaptive"},
		{Label: "flow", Key: "F", Value: "gated"},
	}, columns, width)

	if len(rows) != 2 {
		t.Fatalf("four fields in four columns drew %d rows", len(rows))
	}

	values := ansi.Strip(rows[1])
	if strings.Contains(values, "highhigh") {
		t.Errorf("the effort ran into the engine: %q", values)
	}

	// Every value is still there, and each is followed by space rather
	// than by the next one.
	for _, want := range []string{"high", "adaptive", "gated"} {
		if !strings.Contains(values, want) {
			t.Errorf("%q is not in the row: %q", want, values)
		}
	}
}

// TestAValueTooLongForItsCellIsCutAndStillSeparated.
func TestAValueTooLongForItsCellIsCutAndStillSeparated(t *testing.T) {
	rows := Fields([]Field{
		{Label: "engine", Value: strings.Repeat("x", 80)},
		{Label: "effort", Value: "high"},
	}, 2, 40)

	values := ansi.Strip(rows[1])
	if !strings.Contains(values, "…") {
		t.Errorf("a value wider than its cell was not cut: %q", values)
	}

	if strings.Contains(values, "…high") {
		t.Errorf("the cut value touches the next column: %q", values)
	}
}
