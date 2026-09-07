package cells

// What a terminal draws is not what a string counts, which is the whole
// reason this package exists: the cases below are the ones that were wrong
// when the window measured in bytes.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// TestFitCutsInCellsAndNotInBytes.
func TestFitCutsInCellsAndNotInBytes(t *testing.T) {
	// The contract is a line no wider than the room, never a line padded
	// out to it: what fills a row is the caller's business, and a wide rune
	// that would land half in and half out is dropped whole.
	for _, c := range []struct {
		name string
		text string
		w    int
		want string
	}{
		{name: "already fits", text: "hola", w: 10, want: "hola"},
		{name: "an accent is one cell", text: "árbol", w: 5, want: "árbol"},
		{name: "cut carries an ellipsis", text: "hola mundo", w: 6, want: "hola …"},
		{name: "a wide rune is two cells", text: "日本語", w: 4, want: "日…"},
		{name: "no room at all", text: "hola", w: 0, want: ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := Fit(c.text, c.w)
			if got != c.want {
				t.Errorf("Fit(%q, %d) = %q, want %q", c.text, c.w, got, c.want)
			}

			if w := ansi.StringWidth(got); w > c.w {
				t.Errorf("Fit(%q, %d) came back %d cells wide", c.text, c.w, w)
			}
		})
	}
}

// TestFitKeepsTheStyleWhole. A cut that lands inside an escape sequence
// leaves the terminal painting everything after it in that colour.
func TestFitKeepsTheStyleWhole(t *testing.T) {
	painted := "\x1b[31mhola mundo\x1b[0m"

	got := Fit(painted, 6)
	if strings.Count(got, "\x1b[") == 0 {
		t.Fatalf("Fit dropped the styling: %q", got)
	}

	if !strings.HasSuffix(got, "\x1b[0m") && !strings.Contains(got, "\x1b[0m") {
		t.Errorf("Fit(%q) left the style open: %q", painted, got)
	}
}

// TestFillTakesTheHeightItWasGiven, and never more: a pane that answers
// eleven rows for ten is a pane that pushes the row under it off the screen.
func TestFillTakesTheHeightItWasGiven(t *testing.T) {
	for _, c := range []struct {
		name  string
		lines []string
		h     int
	}{
		{name: "short of the height", lines: []string{"a", "b"}, h: 5},
		{name: "over the height", lines: []string{"a", "b", "c"}, h: 2},
		{name: "exactly the height", lines: []string{"a", "b"}, h: 2},
		{name: "no height at all", lines: []string{"a"}, h: 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := Fill(c.lines, c.h); len(got) != c.h {
				t.Errorf("Fill(%d lines, %d) gave %d rows", len(c.lines), c.h, len(got))
			}
		})
	}
}

// TestPadRightMeasuresCellsToo.
func TestPadRightMeasuresCellsToo(t *testing.T) {
	if got := PadRight("hola", 8); ansi.StringWidth(got) != 8 {
		t.Errorf("PadRight(%q, 8) = %q", "hola", got)
	}

	if got := PadRight("día", 3); got != "día" {
		t.Errorf("PadRight left a string already that wide alone: %q", got)
	}

	if got := PadRight("hola mundo", 4); got != "hola mundo" {
		t.Errorf("PadRight cut a string wider than the width: %q", got)
	}
}

// TestLinesBreakAtWordsAndThenAnywhere. A word wider than the room has to be
// broken somewhere, and a paragraph that refuses to break it draws past the
// edge of the box.
func TestLinesBreakAtWordsAndThenAnywhere(t *testing.T) {
	rows := Lines("uno dos tres cuatro", 8)
	for _, row := range rows {
		if w := ansi.StringWidth(row); w > 8 {
			t.Errorf("a row is %d cells wide: %q", w, row)
		}
	}

	if len(rows) < 2 {
		t.Errorf("a paragraph wider than the room stayed on one row: %q", rows)
	}

	long := Lines("supercalifragilistico", 6)
	if len(long) < 2 {
		t.Errorf("a word wider than the room was not broken: %q", long)
	}
}

// TestTheSmallAnswers is the handful of one-line questions the window asks
// about a value it is about to draw.
func TestTheSmallAnswers(t *testing.T) {
	if got := OrDef("", "en"); got != "en" {
		t.Errorf("OrDef of nothing = %q", got)
	}

	if got := OrDef("es", "en"); got != "es" {
		t.Errorf("OrDef of a value = %q", got)
	}

	if got := First(nil); got != "" {
		t.Errorf("First of no list = %q", got)
	}

	if got := First([]string{"agy", "claude"}); got != "agy" {
		t.Errorf("First = %q", got)
	}

	// Runes, not bytes: a backspace on "día" leaves "dí".
	if got := TrimLastRune("día"); got != "dí" {
		t.Errorf("TrimLastRune(%q) = %q", "día", got)
	}

	if got := TrimLastRune(""); got != "" {
		t.Errorf("TrimLastRune of nothing = %q", got)
	}
}

// TestDialLabelFallsBackToTheOptionItself. A dial with no labels of its own
// draws what it holds, which is what every dial but the model does.
func TestDialLabelFallsBackToTheOptionItself(t *testing.T) {
	ids := []string{"", "opencode/sonnet"}
	labels := []string{"default", "sonnet"}

	if got := DialLabel(ids, labels, 1); got != "sonnet" {
		t.Errorf("DialLabel with labels = %q", got)
	}

	if got := DialLabel(ids, nil, 1); got != "opencode/sonnet" {
		t.Errorf("DialLabel without labels = %q", got)
	}

	if got := DialLabel(ids, labels, 9); got != "" {
		t.Errorf("DialLabel past the end = %q", got)
	}
}

// TestALineTheWriterEndedIsALine. These fields hold paragraphs, and wrapping
// them as one run of words joins a list of checks into a sentence.
func TestALineTheWriterEndedIsALine(t *testing.T) {
	got := WrapKeeping("make check\nmake coverage\ngo test -race ./...", 40)
	if len(got) != 3 {
		t.Fatalf("three lines wrapped to %d: %q", len(got), got)
	}

	for i, want := range []string{"make check", "make coverage", "go test -race ./..."} {
		if got[i] != want {
			t.Errorf("line %d is %q, want %q", i, got[i], want)
		}
	}

	// A paragraph longer than the measure is folded, and the words are kept
	// whole.
	long := WrapKeeping("the webhook retries on 5xx and gives up after the fifth attempt", 20)
	if len(long) < 3 {
		t.Fatalf("a long paragraph wrapped to %d lines: %q", len(long), long)
	}

	for _, l := range long {
		if lipgloss.Width(l) > 20 {
			t.Errorf("a wrapped line is %d cells wide: %q", lipgloss.Width(l), l)
		}
	}

	// Nothing to wrap, and nowhere to wrap it into, are both nothing.
	if got := WrapKeeping("   ", 20); got != nil {
		t.Errorf("a blank paragraph wrapped to %q", got)
	}

	if got := WrapKeeping("something", 0); got != nil {
		t.Errorf("wrapping into no room drew %q", got)
	}
}

// TestABlankLineSurvivesTheWrap, because a paragraph break is what the
// writer meant by it.
func TestABlankLineSurvivesTheWrap(t *testing.T) {
	got := WrapKeeping("first\n\nsecond", 40)
	if len(got) != 3 || got[1] != "" {
		t.Errorf("a paragraph break wrapped to %q", got)
	}
}

// TestADialComesRoundAtBothEnds, so a reader holding a key down never finds
// an end to it.
func TestADialComesRoundAtBothEnds(t *testing.T) {
	opts := []string{"low", "medium", "high"}

	for _, c := range []struct {
		current string
		delta   int
		want    string
	}{
		{"low", 1, "medium"},
		{"high", 1, "low"},
		{"low", -1, "high"},
		{"medium", -1, "low"},
		{"nothing anybody offered", 1, "medium"},
		{"low", 0, "low"},
	} {
		if got := NextOption(opts, c.current, c.delta); got != c.want {
			t.Errorf("NextOption(%q, %d) = %q, want %q", c.current, c.delta, got, c.want)
		}
	}

	// A dial with nothing on it stays where it is rather than reaching into
	// an empty list.
	if got := NextOption(nil, "low", 1); got != "low" {
		t.Errorf("a dial with no options turned to %q", got)
	}
}
