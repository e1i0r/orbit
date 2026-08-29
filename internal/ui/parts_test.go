package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// TestACardIsExactlyAsWideAsAsked. Cards are set side by side, so one that
// came back a column short of its share would shift every card to its right
// and leave the strip's own right edge ragged.
func TestACardIsExactlyAsWideAsAsked(t *testing.T) {
	for _, w := range []int{cardFloor, 20, 41, 140} {
		for _, l := range card("cost", []string{"$1.20"}, w) {
			if got := lipgloss.Width(l); got != w {
				t.Errorf("card(width %d) drew a line %d cells wide", w, got)
			}
		}
	}
}

// TestACardTooNarrowToDrawIsDrawnAtTheFloor. A pane can be any width the
// terminal is, and a card asked for less than its own chrome would otherwise
// be a border with a negative body.
func TestACardTooNarrowToDrawIsDrawnAtTheFloor(t *testing.T) {
	for _, w := range []int{-4, 0, cardFloor - 1} {
		lines := card("", []string{"x"}, w)
		if got := lipgloss.Width(lines[0]); got != cardFloor {
			t.Errorf("card(width %d) drew %d cells wide, want the floor of %d", w, got, cardFloor)
		}
	}
}

// TestACardKeepsItsBodyOffItsBorder. A body line longer than the card wraps
// unless it is cut first, and a wrapped card is a card one line taller than
// the cards beside it.
func TestACardKeepsItsBodyOffItsBorder(t *testing.T) {
	long := strings.Repeat("no", 60)

	lines := card("a title far too long to fit inside this card", []string{long}, 20)
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
func TestFieldsSetTheLabelAboveTheValue(t *testing.T) {
	pairs := []field{
		{"engine [k]", "claude sonnet"},
		{"effort [E]", "high"},
		{"thinking [t]", "adaptive"},
	}

	got := fields(pairs, 2, 60)
	if len(got) != 5 {
		t.Fatalf("fields laid 3 pairs in 2 columns as %d lines, want two rows and a blank between them", len(got))
	}

	if got[2] != "" {
		t.Errorf("the line between two rows is %q, want a blank one", got[2])
	}

	for i, want := range []string{"ENGINE [K]", "claude sonnet", "THINKING [T]", "adaptive"} {
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
	pairs := []field{{"a", "1"}, {"bbbbbbbbbbbbbbbbbbbb", "2"}, {"c", "3"}}

	got := fields(pairs, 3, 60)
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
	if got := fields([]field{{"a", "1"}}, 0, 60); got != nil {
		t.Errorf("fields in 0 columns returned %q", got)
	}

	if got := fields(nil, 2, 60); got != nil {
		t.Errorf("fields with no pairs returned %q", got)
	}
}

// TestABadgeIsItsRoleOnItsOwnTint, which is the whole point of a badge: a
// pane can carry several without any of them shouting.
func TestABadgeIsItsRoleOnItsOwnTint(t *testing.T) {
	seen := map[string]Role{}

	for _, r := range []Role{OK, Bad, Warn, Live, Accent} {
		got := badge("done", r)
		if !strings.Contains(got, "done") {
			t.Errorf("badge(%v) lost its text: %q", r, got)
		}

		if was, dup := seen[got]; dup {
			t.Errorf("badge(%v) renders exactly like badge(%v)", r, was)
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
			plain, rendered := tabChip(c.key, c.text, active)

			if !strings.Contains(plain, c.key) || !strings.Contains(ansi.Strip(rendered), c.key) {
				t.Errorf("tabChip(%q, %q, %v) = %q / %q, want the key in both", c.key, c.text, active, plain, rendered)
			}

			if c.text != "" && !strings.Contains(plain, c.text) {
				t.Errorf("tabChip(%q, %q, %v) lost the name: %q", c.key, c.text, active, plain)
			}

			if got := ansi.Strip(rendered); got != plain {
				t.Errorf("tabChip(%q, %q, %v) drew %q but reports %q, and the strip is clicked by the report",
					c.key, c.text, active, got, plain)
			}
		}
	}
}

// TestTheOpenTabIsUnderlined. Underline is what says which of eleven chips is
// the one being read; the others are told apart by it being absent.
func TestTheOpenTabIsUnderlined(t *testing.T) {
	_, open := tabChip("1", "overview", true)
	_, shut := tabChip("1", "overview", false)

	if with, runs := underlinedRuns(open); with != runs || runs == 0 {
		t.Errorf("the open tab has %d of %d runs underlined: %q", with, runs, open)
	}

	if with, _ := underlinedRuns(shut); with != 0 {
		t.Errorf("a closed tab has %d runs underlined: %q", with, shut)
	}
}

// underlinedRuns counts the escape sequences in s that set a style, and how
// many of those also set SGR 4. The key and the name are styled separately,
// so an underline under one of them and not the other is a rule that stops
// half way across the chip.
//
// It walks the parameters rather than searching the text for a 4, because a
// truecolour sequence spells a colour out in decimal and any of those five
// numbers can be a 4 that means nothing about underlining.
func underlinedRuns(s string) (with, runs int) {
	for _, seq := range strings.Split(s, "\x1b[") {
		end := strings.Index(seq, "m")
		if end < 1 {
			continue // no sequence here, or the reset, which styles nothing
		}

		runs++

		if setsUnderline(strings.Split(seq[:end], ";")) {
			with++
		}
	}

	return with, runs
}

// setsUnderline reports whether one sequence's parameters include SGR 4,
// skipping the five a truecolour takes: any of a colour's decimal channels
// can be a 4 that says nothing about underlining.
func setsUnderline(params []string) bool {
	for i := 0; i < len(params); i++ {
		switch params[i] {
		case "38", "48":
			i += 4 // 38;2;r;g;b, which is what lipgloss writes here
		case "4":
			return true
		}
	}

	return false
}

// thumbRun is where the thumb sits on a track and how much of it it covers.
func thumbRun(col []string) (first, last, rows int) {
	first, last = -1, -1

	for i, c := range col {
		if !strings.Contains(c, scrollThumb) {
			continue
		}

		if first < 0 {
			first = i
		}

		last, rows = i, rows+1
	}

	return first, last, rows
}

// TestAPaneThatFitsHasNoBar. A rail that cannot move says there is somewhere
// to go, and spends a column of every pane saying it.
func TestAPaneThatFitsHasNoBar(t *testing.T) {
	for _, c := range []struct{ rows, total int }{{10, 10}, {10, 3}, {0, 40}} {
		if got := scrollTrack(c.rows, c.total, 0); got != nil {
			t.Errorf("scrollTrack(%d rows, %d lines) drew %d rows of bar", c.rows, c.total, len(got))
		}
	}
}

// TestTheBarIsOneCellWide. The rail is drawn in the pane's last column over
// a line filled to the column before it, so a two-cell rail wraps every row
// of the pane onto a second line.
func TestTheBarIsOneCellWide(t *testing.T) {
	for _, c := range scrollTrack(12, 90, 30) {
		if got := lipgloss.Width(c); got != 1 {
			t.Errorf("a row of the bar is %d cells wide, want 1", got)
		}
	}
}

// TestTheThumbSaysWhereTheReaderIs. Top of the text, top of the rail; end of
// the text, floor of the rail. Dividing down leaves the thumb a cell short
// of the floor on the last screen, which reads as more to come.
func TestTheThumbSaysWhereTheReaderIs(t *testing.T) {
	const rows, total = 10, 90

	first, _, _ := thumbRun(scrollTrack(rows, total, 0))
	if first != 0 {
		t.Errorf("at the top of the text the thumb starts on row %d, want 0", first)
	}

	_, last, _ := thumbRun(scrollTrack(rows, total, total-rows))
	if last != rows-1 {
		t.Errorf("at the end of the text the thumb ends on row %d, want %d", last, rows-1)
	}
}

// TestTheThumbIsTheShareThatShows. How tall the thumb is, is how much of the
// text is on the screen: a tenth of it is a tenth of the rail, and never
// nothing at all.
func TestTheThumbIsTheShareThatShows(t *testing.T) {
	for _, c := range []struct{ rows, total, want int }{
		{10, 20, 5},
		{10, 100, 1},
		{10, 40000, 1},
		{20, 30, 13},
	} {
		if _, _, got := thumbRun(scrollTrack(c.rows, c.total, 0)); got != c.want {
			t.Errorf("%d rows over %d lines: thumb is %d rows, want %d", c.rows, c.total, got, c.want)
		}
	}
}
