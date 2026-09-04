package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// headerCell is the column where s begins on the header line as it is
// actually drawn, measured the way a terminal measures: in cells, counting
// each emoji ahead of it as the two it takes.
//
// Every test below asks for a position this way rather than writing one
// down. A column written down in a test is the same mistake as a column
// written down in hitHeader, one file over.
func headerCell(t *testing.T, m Model, s string) int {
	t.Helper()

	plain := ansi.Strip(m.headerLine(m.frame.Header.W))

	i := strings.Index(plain, s)
	if i < 0 {
		t.Fatalf("the header line does not say %q: %q", s, plain)
	}

	return lipgloss.Width(plain[:i])
}

// TestClickingAQueueBadgeFiltersByTheQueueDrawnThere.
//
// Answering from eight column numbers of hitHeader's own — under 10 is the
// badge, 28 to 44 is Running — puts them a badge and a half out as soon as
// the header is laid out again: the left third of Running filters by To Do,
// and clicking Needs You filters by Running.
func TestClickingAQueueBadgeFiltersByTheQueueDrawnThere(t *testing.T) {
	m, _ := testModel(t, 150, 30)
	y := m.frame.HeaderLineY()

	for _, c := range []struct {
		icon, label string
		want        view.Band
	}{
		{"📋", "To Do", view.ToDo},
		{"⚡", "Running", view.Running},
		{"💬", "Needs You", view.NeedsYou},
		{"🏁", "Done", view.Done},
	} {
		// The badge's own left edge, which is the padding cell before its
		// icon, and the last letter of its name.
		edge := headerCell(t, m, c.icon) - 1
		last := headerCell(t, m, c.label) + lipgloss.Width(c.label) - 1

		for _, x := range []int{edge, last} {
			got := m.hitHeader(x, y)
			if got.Kind != TargetHeaderQueue || got.Band != c.want {
				t.Errorf("hitHeader(%d) = %+v, want the %v badge, which is what is drawn there",
					x, got, c.want)
			}
		}
	}
}

// TestSelectingAQueueDoesNotMoveTheOthersOutFromUnderTheCursor.
//
// The badge of the queue being filtered on is marked, and the mark makes it
// two cells wider, so choosing one queue shifts every badge to its right.
// Fixed columns cannot follow that: a reader who filtered by Running and
// then went to click Done landed two cells short of it.
func TestSelectingAQueueDoesNotMoveTheOthersOutFromUnderTheCursor(t *testing.T) {
	m, _ := testModel(t, 150, 30)
	y := m.frame.HeaderLineY()

	band := view.Running
	m.queueFilter = &band

	// Done's own left edge, two cells further right than it was before
	// Running grew its mark.
	x := headerCell(t, m, "🏁") - 1
	if got := m.hitHeader(x, y); got.Kind != TargetHeaderQueue || got.Band != view.Done {
		t.Errorf("with Running selected, hitHeader(%d) = %+v, want Done", x, got)
	}
}

// TestClickingAStandingFactOpensTheFactDrawnThere. The three chips on the
// right were placed by counting back from the terminal's edge, which is only
// right for as long as none of them changes width — and the repo count, the
// engine name and the language all do.
func TestClickingAStandingFactOpensTheFactDrawnThere(t *testing.T) {
	m, _ := testModel(t, 150, 30)
	y := m.frame.HeaderLineY()

	for _, c := range []struct {
		icon, tail, field string
	}{
		{"📦", "repos", "repos"},
		{"🧠", "claude", "engine"},
		{"🌐", "EN", "lang"},
	} {
		// The chip's icon, and the last letter of the word it ends on.
		for _, x := range []int{
			headerCell(t, m, c.icon),
			headerCell(t, m, c.tail) + lipgloss.Width(c.tail) - 1,
		} {
			got := m.hitHeader(x, y)
			if got.Kind != TargetHeaderField || got.Field != c.field {
				t.Errorf("hitHeader(%d) = %+v, want the %s field, which is what is drawn there",
					x, got, c.field)
			}
		}
	}
}

// TestTheBlankCellsOfTheHeaderAreInert. The spaces between one chip and the
// next are not a near miss of either: nothing is drawn there, so nothing
// happens there. Columns counted back from the terminal's edge had no way to
// know that, and handed every gap to whichever chip was nearest.
func TestTheBlankCellsOfTheHeaderAreInert(t *testing.T) {
	m, _ := testModel(t, 150, 30)
	y := m.frame.HeaderLineY()

	for _, icon := range []string{"📦", "🧠", "🌐"} {
		x := headerCell(t, m, icon) - 1
		if got := m.hitHeader(x, y); got.Kind != TargetNone {
			t.Errorf("hitHeader(%d) = %+v, want nothing: the cell before %s is blank", x, got, icon)
		}
	}
}
