package flows

// Pointing at an option chooses that option.
//
// The designer draws its short dials as a row of pills, and a click on one
// of them chose nothing: the hit-test answered the row, and the row's own
// action steps the dial one along. So clicking "off" on a dial sitting on
// "adaptive" moved it to "on" — the reader clicked again, and watched.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// designing is the builder open on a flow, in a window with room for it.
func designing(t *testing.T, w, h int) (State, Env) {
	t.Helper()

	e := world(t)

	frame, err := layout.Fit(w, h)
	if err != nil {
		t.Fatalf("a window of %dx%d will not fit: %v", w, h, err)
	}

	e.Frame = frame

	s, _ := Open(FromBoard, e).editFlow("careful", e)
	if !s.Creating() {
		t.Fatal("the designer did not open")
	}

	return s, e
}

// TestEveryPillOfADialIsTheOptionUnderIt. Every column of every pill
// answers with its own option, the gap between two answers neither, and
// what the pointer is over is what the row is drawn as.
func TestEveryPillOfADialIsTheOptionUnderIt(t *testing.T) {
	s, e := designing(t, 120, 45)

	for _, field := range []int{
		flowFieldTemplate, flowFieldEngine, flowFieldThinking,
		flowFieldFeedOutput, flowFieldWait, flowFieldIsLoop,
	} {
		opts, current, ok := s.choices(field, e)
		if !ok {
			t.Fatalf("field %d offers nothing", field)
		}

		row := rowOfField(t, s, field, e)
		at := valueAt

		for i, o := range opts {
			wide := lipgloss.Width(o.label)
			if o.id == current {
				wide += 2
			}

			for _, x := range []int{at, at + wide - 1} {
				got := s.Hit(x, row, e)
				if got.Field != "dial" || got.Phase != field || got.Pane != i {
					t.Errorf("field %d, column %d is %q phase=%d option=%d, want option %d",
						field, x, got.Field, got.Phase, got.Pane, i)
				}
			}

			if i+1 < len(opts) {
				if got := s.Hit(at+wide, row, e); got.Field == "dial" {
					t.Errorf("field %d, the gap at column %d answers option %d", field, at+wide, got.Pane)
				}
			}

			at += wide + 1
		}
	}
}

// TestClickingAnOptionSetsThatOption, and not the one after the one that
// was already on.
func TestClickingAnOptionSetsThatOption(t *testing.T) {
	s, e := designing(t, 120, 45)

	for _, c := range []struct {
		field int
		want  int
		says  func(State) string
	}{
		// "off" is two along from "adaptive": stepping one would land on
		// "on", which is what made this worth writing down.
		{flowFieldThinking, 2, func(s State) string { return s.edited().Thinking }},
		{flowFieldThinking, 1, func(s State) string { return s.edited().Thinking }},
		{flowFieldWait, 1, func(s State) string { return pickOne(s.cur().Wait, "no", "yes") }},
		{flowFieldFeedOutput, 0, func(s State) string { return pickOne(s.edited().FeedOutput, "no", "yes") }},
		{flowFieldIsLoop, 1, func(s State) string { return pickOne(s.looping(), "no", "yes") }},
	} {
		opts, _, _ := s.choices(c.field, e)

		next, _ := s.Click(point.Target{
			Kind: point.FlowItem, Field: "dial", Phase: c.field, Pane: c.want,
		}, e)

		if next.field != c.field {
			t.Errorf("clicking a pill of field %d left the cursor on field %d", c.field, next.field)
		}

		gotOpts, gotCurrent, _ := next.choices(c.field, e)
		if gotCurrent != gotOpts[c.want].id {
			t.Errorf("clicking option %d of field %d (%q) left it on %q",
				c.want, c.field, opts[c.want].id, gotCurrent)
		}
	}
}

// TestAPillOfTheDialIsWhereTheDialIsDrawn. The row is laid out by one
// function and measured by another, and they agree only while both read the
// same list of options — which is why there is one.
func TestAPillOfTheDialIsWhereTheDialIsDrawn(t *testing.T) {
	s, e := designing(t, 120, 45)

	for _, field := range []int{flowFieldThinking, flowFieldWait, flowFieldFeedOutput} {
		opts, current, _ := s.choices(field, e)
		row := rowOfField(t, s, field, e)

		lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)
		drawn := ansi.Strip(lines[start+row-e.Frame.Body.Y].text)

		for _, o := range opts {
			if !strings.Contains(drawn, o.label) {
				t.Errorf("field %d draws %q, which does not say %q", field, drawn, o.label)
			}
		}

		// The pill the pointer is over is the word at that column.
		at := valueAt

		for _, o := range opts {
			wide := lipgloss.Width(o.label)
			if o.id == current {
				wide += 2
			}

			under := strings.TrimSpace(cellsAt(drawn, at, wide))
			if under != strings.TrimSpace(o.label) {
				t.Errorf("field %d: columns %d..%d draw %q, and the pointer calls them %q",
					field, at, at+wide-1, under, o.label)
			}

			at += wide + 1
		}
	}
}

// TestThePhaseBeingEditedIsChosenByPointingAtIt, the same gesture the
// pipeline above the form already answers.
func TestThePhaseBeingEditedIsChosenByPointingAtIt(t *testing.T) {
	s, e := designing(t, 120, 45)

	if len(s.phases) < 3 {
		t.Fatalf("this test needs three phases, and the flow has %d", len(s.phases))
	}

	row := rowOfField(t, s, flowFieldPhaseSelect, e)
	at := valueAt

	for i, label := range s.phaseLabels() {
		wide := lipgloss.Width(label) + 2
		if i == s.activePhase {
			wide += 2
		}

		got := s.Hit(at, row, e)
		if got.Field != "select_phase" || got.Pane != 0 || got.Phase != i {
			t.Errorf("column %d of the phase row is %q phase=%d, want phase %d", at, got.Field, got.Phase, i)
		}

		at += wide + 1
	}

	// And clicking the last one edits the last one.
	last := len(s.phases) - 1

	next, _ := s.Click(point.Target{Kind: point.FlowItem, Field: "select_phase", Phase: last}, e)
	if next.activePhase != last {
		t.Errorf("clicking phase %d left the designer editing phase %d", last, next.activePhase)
	}
}

// cellsAt is the slice of a drawn row between two columns.
func cellsAt(row string, from, wide int) string {
	rs := []rune(row)
	if from >= len(rs) {
		return ""
	}

	return string(rs[from:min(from+wide, len(rs))])
}
