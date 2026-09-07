package prose

// The shapes a screen is built out of, checked at the widths a terminal
// actually has.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

// TestACardIsExactlyAsWideAsAsked. Cards are set side by side, so one that
// came back a column short of its share would shift every card to its right
// and leave the strip's own right edge ragged.
func TestACardIsExactlyAsWideAsAsked(t *testing.T) {
	for _, w := range []int{cardFloor, 20, 41, 140} {
		for _, l := range Card("cost", []string{"$1.20"}, w) {
			if got := lipgloss.Width(l); got != w {
				t.Errorf("Card(width %d) drew a line %d cells wide", w, got)
			}
		}
	}
}

// TestACardTooNarrowToDrawIsDrawnAtTheFloor. A pane can be any width the
// terminal is, and a card asked for less than its own chrome would otherwise
// be a border with a negative body.
func TestACardTooNarrowToDrawIsDrawnAtTheFloor(t *testing.T) {
	for _, w := range []int{-4, 0, cardFloor - 1} {
		lines := Card("", []string{"x"}, w)
		if got := lipgloss.Width(lines[0]); got != cardFloor {
			t.Errorf("Card(width %d) drew %d cells wide, want the floor of %d", w, got, cardFloor)
		}
	}
}

// TestACardKeepsItsBodyOffItsBorder. A body line longer than the card wraps
// unless it is cut first, and a wrapped card is a card one line taller than
// the cards beside it.
func TestACardKeepsItsBodyOffItsBorder(t *testing.T) {
	long := strings.Repeat("no", 60)

	lines := Card("a title far too long to fit inside this card", []string{long}, 20)
	if len(lines) != 4 {
		t.Fatalf("card wrapped into %d lines, want a top, a title, a body and a bottom", len(lines))
	}

	for _, l := range lines {
		if got := lipgloss.Width(l); got != 20 {
			t.Errorf("card line is %d cells wide, want 20: %q", got, l)
		}
	}
}

// TestFieldsSetTheLabelAboveTheValue, in that order, because the pair is read
// as a caption and the thing it captions rather than as two values.
//
// The label is upper-cased and the key beside it is not. Written into the
// label, thinking's t was drawn as [T] — which on that screen is the key for
// more tests, and pressing what the pane named queued an engine run.
func TestFieldsSetTheLabelAboveTheValue(t *testing.T) {
	pairs := []Field{
		{Label: "engine", Key: "k", Value: "claude sonnet"},
		{Label: "effort", Key: "E", Value: "high"},
		{Label: "thinking", Key: "t", Value: "adaptive"},
	}

	got := Fields(pairs, 2, 60)
	if len(got) != 5 {
		t.Fatalf("fields laid 3 pairs in 2 columns as %d lines, want two rows and a blank between them", len(got))
	}

	if got[2] != "" {
		t.Errorf("the line between two rows is %q, want a blank one", got[2])
	}

	for i, want := range []string{"ENGINE [k]", "claude sonnet", "THINKING [t]", "adaptive"} {
		line := got[[]int{0, 1, 3, 4}[i]]
		if !strings.Contains(line, want) {
			t.Errorf("line %d is %q, want it to state %q", i, line, want)
		}
	}

	if !strings.Contains(got[0], "EFFORT [E]") {
		t.Errorf("the second column is missing from the first row: %q", got[0])
	}
}

// TestFieldsHoldTheirColumns. The value under a label is the value of that
// label, and it only reads that way while both start in the same cell.
func TestFieldsHoldTheirColumns(t *testing.T) {
	pairs := []Field{
		{Label: "a", Value: "1"},
		{Label: "bbbbbbbbbbbbbbbbbbbb", Value: "2"},
		{Label: "c", Value: "3"},
	}

	got := Fields(pairs, 3, 60)
	if len(got) != 2 {
		t.Fatalf("fields returned %d lines, want a label line and a value line", len(got))
	}

	labels, values := ansi.Strip(got[0]), ansi.Strip(got[1])

	for _, c := range []struct{ label, value string }{{"A", "1"}, {"BBBB", "2"}, {"C", "3"}} {
		labelAt, valueAt := strings.Index(labels, c.label), strings.Index(values, c.value)
		if labelAt != valueAt {
			t.Errorf("%s starts at cell %d and its value at %d", c.label, labelAt, valueAt)
		}
	}
}

// TestFieldsRefuseWhatTheyCannotLay. Zero columns is a division by zero and
// no pairs is a blank line where a block was expected.
func TestFieldsRefuseWhatTheyCannotLay(t *testing.T) {
	if got := Fields([]Field{{Label: "a", Value: "1"}}, 0, 60); got != nil {
		t.Errorf("fields in 0 columns returned %q", got)
	}

	if got := Fields(nil, 2, 60); got != nil {
		t.Errorf("fields with no pairs returned %q", got)
	}
}

// TestABadgeIsItsRoleOnItsOwnTint, which is the whole point of a badge: a
// pane can carry several without any of them shouting.
func TestABadgeIsItsRoleOnItsOwnTint(t *testing.T) {
	seen := map[string]theme.Role{}

	for _, r := range []theme.Role{theme.OK, theme.Bad, theme.Warn, theme.Live, theme.Accent} {
		got := Badge("done", r)
		if !strings.Contains(got, "done") {
			t.Errorf("Badge(%v) lost its text: %q", r, got)
		}

		if was, dup := seen[got]; dup {
			t.Errorf("Badge(%v) renders exactly like Badge(%v)", r, was)
		}

		seen[got] = r
	}
}

// TestATabChipNamesItsKey, whichever tier the strip is drawn at. The number
// is how a tab is opened, so a chip that dropped it would leave the strip
// saying what the tabs are called and nothing about how to get to one.
func TestATabChipNamesItsKey(t *testing.T) {
	for _, c := range []struct{ key, text string }{{"6", "timeline"}, {"w", "thinking"}, {"0", ""}} {
		for _, active := range []bool{false, true} {
			plain, rendered := Chip(c.key, c.text, active)

			if !strings.Contains(plain, c.key) || !strings.Contains(ansi.Strip(rendered), c.key) {
				t.Errorf("Chip(%q, %q, %v) = %q / %q, want the key in both", c.key, c.text, active, plain, rendered)
			}

			if c.text != "" && !strings.Contains(plain, c.text) {
				t.Errorf("Chip(%q, %q, %v) lost the name: %q", c.key, c.text, active, plain)
			}

			if got := ansi.Strip(rendered); got != plain {
				t.Errorf("Chip(%q, %q, %v) drew %q but reports %q, and the strip is clicked by the report",
					c.key, c.text, active, got, plain)
			}
		}
	}
}

// TestTheOpenTabIsABand. The band is what says which of eleven chips is the
// one being read; the others are told apart by it being absent.
func TestTheOpenTabIsABand(t *testing.T) {
	openPlain, open := Chip("1", "overview", true)
	_, shut := Chip("1", "overview", false)

	if with, runs := bandedRuns(open); with != runs || runs == 0 {
		t.Errorf("the open tab has %d of %d runs behind a band: %q", with, runs, open)
	}

	if with, _ := bandedRuns(shut); with != 0 {
		t.Errorf("a closed tab has %d runs behind a band: %q", with, shut)
	}

	// The band covers a cell either side of the name rather than sitting
	// tight against it, and the padding is reported so a click on it lands
	// on the tab it is drawn over.
	if !strings.HasPrefix(openPlain, " ") || !strings.HasSuffix(openPlain, " ") {
		t.Errorf("the open tab reports %q, want the pad the band is drawn over", openPlain)
	}
}

// bandedRuns counts the escape sequences in s that set a style, and how many
// of those also set a background. The key and the name can be styled
// separately, so a band behind one of them and not the other is a mark that
// stops half way across the chip.
//
// It walks the parameters rather than searching the text for a 48, because a
// truecolour sequence spells a colour out in decimal and any of those five
// numbers can be a 48 that means nothing about a background.
func bandedRuns(s string) (with, runs int) {
	for _, seq := range strings.Split(s, "\x1b[") {
		end := strings.Index(seq, "m")
		if end < 1 {
			continue // no sequence here, or the reset, which styles nothing
		}

		runs++

		if setsBackground(strings.Split(seq[:end], ";")) {
			with++
		}
	}

	return with, runs
}

// setsBackground reports whether one sequence's parameters include SGR 48,
// skipping the five a truecolour takes: any of a foreground's decimal
// channels can be a 48 that says nothing about a background.
func setsBackground(params []string) bool {
	for i := 0; i < len(params); i++ {
		switch params[i] {
		case "48":
			return true
		case "38":
			i += 4 // 38;2;r;g;b, which is what lipgloss writes here
		}
	}

	return false
}
