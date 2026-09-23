package flows

// The three tabs of the designer, and where a click on one lands.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/point"
)

// TestEveryTabAnswersItsOwnColumns, and the hint beside them answers none.
func TestEveryTabAnswersItsOwnColumns(t *testing.T) {
	s, e := designing(t, 110, 45)

	at := 2

	for i, name := range s.flowTabNames(e) {
		wide := lipgloss.Width(name) + 2

		for _, x := range []int{at, at + wide/2, at + wide - 1} {
			if got := s.flowTabAt(x, e); got != i {
				t.Errorf("column %d of the %q tab is tab %d, want %d", x, name, got, i)
			}
		}

		// The space between two of them belongs to neither.
		if got := s.flowTabAt(at+wide, e); got != -1 {
			t.Errorf("the gap at column %d is tab %d", at+wide, got)
		}

		at += wide + 1
	}

	// And the hint past the last of them is not a tab.
	for _, x := range []int{at, at + 4, 100} {
		if got := s.flowTabAt(x, e); got != -1 {
			t.Errorf("column %d, past the tabs, is tab %d", x, got)
		}
	}

	// Nor is anything to the left of the first.
	for _, x := range []int{0, 1} {
		if got := s.flowTabAt(x, e); got != -1 {
			t.Errorf("column %d, before the tabs, is tab %d", x, got)
		}
	}
}

// TestTheTabBeingReadIsTheOneMarked, and clicking another opens it.
func TestTheTabBeingReadIsTheOneMarked(t *testing.T) {
	s, e := designing(t, 110, 45)

	for _, tab := range []int{flowTabFields, flowTabDiagram, flowTabSay} {
		at := s
		at.tab = tab

		row := ansi.Strip(at.flowTabsRow(e.Frame.Body.W, e).text)

		for i, name := range at.flowTabNames(e) {
			if !strings.Contains(row, name) {
				t.Errorf("with tab %d open the strip does not say %q: %q", tab, name, row)
			}

			_ = i
		}

		// The strip is one row, whatever is on it.
		if strings.Contains(row, "\n") {
			t.Errorf("the tab strip is more than one row: %q", row)
		}
	}

	// A click on a tab opens that tab.
	strip := -1

	lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)
	for i := start; i < len(lines); i++ {
		if lines[i].strip {
			strip = i - start

			break
		}
	}

	if strip < 0 {
		t.Fatal("the tab strip is on no row of the designer")
	}

	at := 2

	for i, name := range s.flowTabNames(e) {
		got := s.Hit(at, e.Frame.Body.Y+strip, e)
		if got.Kind != point.FlowItem || got.Field != "tab" || got.Phase != i {
			t.Errorf("clicking %q answered %q/%d, want tab %d", name, got.Field, got.Phase, i)
		}

		next, _ := s.Click(got, e)
		if next.tab != i {
			t.Errorf("clicking %q opened tab %d", name, next.tab)
		}

		at += lipgloss.Width(name) + 3
	}
}

// TestTheTabStripFitsTheWindow, at every width the designer is drawn at.
func TestTheTabStripFitsTheWindow(t *testing.T) {
	for w := 60; w <= 160; w++ {
		s, e := designing(t, 110, 45)

		if got := lipgloss.Width(s.flowTabsRow(w, e).text); got > w {
			t.Fatalf("at %d columns the tab strip is %d cells wide", w, got)
		}
	}
}
