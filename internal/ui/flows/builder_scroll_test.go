package flows

// The form is taller than a short terminal, and this is the suite that says
// so: which field is on the screen, what the boxes give up before it starts
// scrolling, and the pipeline drawn above it.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/layout"
)

// TestTheFieldBeingEditedIsOnTheScreen, at every height the designer is
// drawn at and on every field it has. Moving between fields brings the
// window back to the one being edited, and a field the window does not hold
// is a reader typing into something they cannot see.
func TestTheFieldBeingEditedIsOnTheScreen(t *testing.T) {
	for _, h := range []int{14, 18, 24, 30, 45, 60} {
		s, e := designing(t, 110, h)

		for _, field := range s.fieldsShown() {
			s.field = field
			s = s.followField(e)

			lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)

			at := s.fieldRow(lines)
			if at < start || at >= start+e.Frame.Body.H {
				t.Fatalf("at %d rows, field %d is on line %d, outside the %d from %d",
					h, field, at, e.Frame.Body.H, start)
			}
		}
	}
}

// TestSaveIsAlwaysReachable. The boxes give their rows up before the form
// starts scrolling — eight rows of instructions are worth less than the
// button that writes them down — and past that the window follows the
// cursor, so the reader who tabs to Save is looking at it.
func TestSaveIsAlwaysReachable(t *testing.T) {
	for _, h := range []int{14, 18, 22, 26, 30, 36, 45} {
		s, e := designing(t, 110, h)

		s.field = flowFieldSave
		s = s.followField(e)

		drawn := ansi.Strip(strings.Join(s.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n"))
		if !strings.Contains(drawn, "Save") {
			t.Errorf("at %d rows, tabbing to Save does not put it on the screen:\n%s", h, drawn)
		}
	}
}

// TestAFormThatFitsIsNotScrolled, and the boxes are what pay for it: the
// same form at the same height is whole once its two boxes have given up
// the rows they can.
func TestAFormThatFitsIsNotScrolled(t *testing.T) {
	for _, h := range []int{30, 36, 45, 60} {
		s, e := designing(t, 110, h)

		lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)
		if len(lines) > e.Frame.Body.H {
			continue // even shrunk, the form is taller than this window
		}

		if start != 0 {
			t.Errorf("at %d rows the form fits in %d lines and still starts at %d", h, len(lines), start)
		}

		drawn := ansi.Strip(strings.Join(s.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n"))
		if !strings.Contains(drawn, "Save") {
			t.Errorf("at %d rows a form that fits does not draw Save:\n%s", h, drawn)
		}
	}
}

// TestNoRowOfTheFormRunsPastTheWindow, at every width and height.
func TestNoRowOfTheFormRunsPastTheWindow(t *testing.T) {
	for w := layout.MinWidth; w <= 150; w += 9 {
		for _, h := range []int{16, 24, 40} {
			s, e := designing(t, w, h)

			for _, tab := range []int{flowTabFields, flowTabDiagram, flowTabSay} {
				at := s
				at.tab = tab

				for i, r := range at.View(e.Frame.Body.H, e.Frame.Body.W, e) {
					if got := lipgloss.Width(r); got > e.Frame.Body.W {
						t.Fatalf("tab %d at %dx%d: row %d is %d cells wide in a body of %d",
							tab, w, h, i, got, e.Frame.Body.W)
					}
				}
			}
		}
	}
}

// TestThePipelineNamesEveryPhaseInOrder, marks the one being edited, and
// draws an arrow between each pair and not before the first.
func TestThePipelineNamesEveryPhaseInOrder(t *testing.T) {
	s, e := designing(t, 110, 45)

	s.phases = []flow.Phase{
		{Name: "implement", Engine: "zeta", Model: "one", Prompt: "write it"},
		{Name: "tests", Engine: "zeta", Loop: &flow.Loop{Max: 4, Phases: []flow.Phase{
			{Name: "inner", Engine: "zeta", Model: "two"},
		}}},
		{Name: "review", Engine: "zeta", Model: "one", Wait: true},
	}
	s.activePhase = 1

	var rows []string

	for _, l := range s.builderPipeline(e.Frame.Body.W, e) {
		rows = append(rows, ansi.Strip(l.text))
	}

	joined := strings.Join(rows, "\n")

	// One row per phase, numbered from one, in the order they run.
	for i, name := range []string{"implement", "tests", "review"} {
		want := "Phase " + string(rune('1'+i)) + ": " + name
		if !strings.Contains(joined, want) {
			t.Errorf("the pipeline does not say %q:\n%s", want, joined)
		}
	}

	// The first phase has nothing before it; the rest are joined.
	first, rest := 0, 0

	for _, r := range rows {
		if !strings.Contains(r, "Phase") {
			continue
		}

		if strings.Contains(r, "➔") {
			rest++

			continue
		}

		first++
	}

	if first != 1 || rest != 2 {
		t.Errorf("%d phases stand alone and %d are joined, want one and two:\n%s", first, rest, joined)
	}

	// Exactly one is marked as the one being edited, and it is the one.
	marked := 0

	for _, r := range rows {
		if strings.Contains(r, "●") {
			marked++

			if !strings.Contains(r, "tests") {
				t.Errorf("the mark is on %q and the cursor is on tests", r)
			}
		}
	}

	if marked != 1 {
		t.Errorf("%d phases are marked, want one:\n%s", marked, joined)
	}

	// A loop says how many turns, and what runs inside it.
	for _, want := range []string{"↻ up to 4×", "zeta/two", "(stops for human)", "write it"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the pipeline does not say %q:\n%s", want, joined)
		}
	}
}

// TestThePipelineHoldsItsQuotedPromptInside, at every width. The line under
// a phase is what it was told, in quotes, and it is indented — a width that
// did not take the indent off runs the quote past the window.
func TestThePipelineHoldsItsQuotedPromptInside(t *testing.T) {
	s, e := designing(t, 110, 45)

	s.phases = []flow.Phase{{
		Name: "implement", Engine: "zeta", Model: "one",
		Prompt: strings.Repeat("a long instruction that keeps going ", 8),
	}}

	for w := 40; w <= 160; w++ {
		for _, l := range s.builderPipeline(w, e) {
			if got := lipgloss.Width(l.text); got > w {
				t.Fatalf("at %d columns a pipeline row is %d cells wide: %q",
					w, got, ansi.Strip(l.text))
			}
		}
	}
}
