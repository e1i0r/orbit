package flows

// flowsmouse_coverage_test.go is wrapPromptText and nextOption's remaining
// branches, and hitFlows's list view — what a click resolves to before the
// builder is open. flowsmouse_builder_coverage_test.go is the builder's own
// half of hitFlows, which has a geometry all its own.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/point"
)

func TestWrapPromptText(t *testing.T) {
	if got := cells.WrapKeeping("", 10); got != nil {
		t.Errorf("cells.WrapKeeping(\"\", 10) = %v, want nil", got)
	}

	if got := cells.WrapKeeping("hello", 0); got != nil {
		t.Errorf("wrapPromptText with maxLen 0 = %v, want nil", got)
	}

	if got := cells.WrapKeeping("   ", 10); got != nil {
		t.Errorf("wrapPromptText of only whitespace = %v, want nil", got)
	}

	if got := cells.WrapKeeping("hello world", 100); len(got) != 1 || got[0] != "hello world" {
		t.Errorf("wrapPromptText that fits on one line = %v", got)
	}

	got := cells.WrapKeeping("aaaaaaaaaa bbbbbbbbbb cccccccccc", 12)
	if len(got) != 3 {
		t.Fatalf("expected 3 wrapped lines, got %d: %v", len(got), got)
	}

	for i, l := range got {
		if l == "" {
			t.Errorf("line %d is empty", i)
		}
	}
}

func TestNextOption(t *testing.T) {
	if got := cells.NextOption(nil, "x", 1); got != "x" {
		t.Errorf("nextOption with no options = %q, want the current value unchanged", got)
	}

	if got := cells.NextOption([]string{"a", "b", "c"}, "not-there", 1); got != "b" {
		t.Errorf("nextOption with an unmatched current = %q, want it to treat the start as index 0", got)
	}

	if got := cells.NextOption([]string{"a", "b", "c"}, "a", -1); got != "c" {
		t.Errorf("nextOption wrapping backward past the start = %q, want c", got)
	}

	if got := cells.NextOption([]string{"a", "b", "c"}, "c", 1); got != "a" {
		t.Errorf("nextOption wrapping forward past the end = %q, want a", got)
	}
}

func TestHitFlowsOutsideBody(t *testing.T) {
	_, e := designer(t)

	s := Open(FromBoard, e)
	if got := s.Hit(10, 0, e); got.Kind != point.None {
		t.Errorf("hitFlows outside the body = %+v, want the zero point.Target", got)
	}
}

func TestHitFlowsListCreateButton(t *testing.T) {
	s, e := designer(t)

	// Found by reading the rows rather than by counting them: the sentence
	// above the button wraps, so which row it lands on is a fact about the
	// width and not a number to write down here.
	row := -1

	for i, l := range s.flowsListLines(e.Frame.Body.W, e) {
		if l.create {
			row = i

			break
		}
	}

	if row < 0 {
		t.Fatal("the create button is on no row of the list")
	}

	got := s.Hit(10, e.Frame.Body.Y+row, e)
	if got.Kind != point.FlowItem || got.Field != "create" {
		t.Errorf("hitFlows at the create row = %+v, want Field \"create\"", got)
	}
}

// TestHitFlowsListEntries walks the list the way flowsRows draws it and
// checks that each flow's own row resolves to a click on that flow.
//
// The rows are found by reading what was drawn rather than by counting lines
// here: a flow is several rows tall and how many depends on its phases, so a
// table of line numbers goes wrong the day any flow changes — and the test
// that fails then is not about what changed.
func TestHitFlowsListEntries(t *testing.T) {
	// Tall enough for every built-in to be on screen at once: the list
	// scrolls, and a row past the bottom is not a row a click can land on.
	s, e := designer(t)
	base := e.Frame.Body.Y

	drawn := s.View(e.Frame.Body.H, e.Frame.Body.W, e)

	for _, name := range flow.BuiltinNames() {
		line := -1

		for i, row := range drawn {
			// The flow's own row is the one that names it and offers the
			// ways into it; its phases are drawn under it and name it not.
			if strings.Contains(ansi.Strip(row), name+"  (") {
				line = i
				break
			}
		}

		if line < 0 {
			t.Errorf("the list does not draw a row for %q", name)
			continue
		}

		got := s.Hit(10, base+line, e)
		if got.Kind != point.FlowItem || got.ID != name {
			t.Errorf("hitFlows on the row of %q = %+v", name, got)
		}
	}

	// Past every listed flow, there is nothing to click. Counted from what
	// was drawn rather than from a line written here, for the reason above.
	if got := s.Hit(10, base+len(drawn)+1, e); got.Kind != point.None {
		t.Errorf("hitFlows past the last flow = %+v, want the zero point.Target", got)
	}
}

// TestHitFlowsListDeleteVsEdit is a reader's own flow, which offers details,
// edit, and delete buttons when selected.
func TestHitFlowsListDeleteVsEdit(t *testing.T) {
	dir := t.TempDir()
	flowJSON := `{"name":"zzz-mine","phases":[{"name":"implement","engine":"claude"}]}`
	writeFlowFile(t, dir, "zzz-mine", flowJSON)

	_, e := designer(t)
	e.Flows = flowsTestDir(dir)
	s := Open(FromBoard, e)
	s.sel = len(flow.BuiltinNames()) // the reader's own, after the builtins
	// The flow's own row, found by reading what was drawn: it is several
	// rows tall and how many depends on its phases, so a line counted here
	// goes wrong the day any flow before it changes.
	base := e.Frame.Body.Y
	line := base + flowRowOf(t, s, "zzz-mine", e)

	// Clicking flow name opens details
	if got := s.Hit(10, line, e); got.Field != "details" || got.ID != "zzz-mine" {
		t.Errorf("hitFlows on flow name = %+v, want a details click", got)
	}

	// The pills, found where they are drawn rather than at two columns
	// written down here. The numbers that used to be here were 44 and 56,
	// and they agreed with a hit-test that measured a pill two cells wider
	// than it is and counted no space between them: both were written from
	// the same wrong idea of the row, and neither was ever compared with it.
	var mine flow.Listed

	for _, d := range flow.List(e.Flows) {
		if d.Name == "zzz-mine" {
			mine = d
		}
	}

	at := rowPillsStart(mine, e)

	for _, pill := range rowPills(mine, e) {
		wide := lipgloss.Width(pill.drawn)

		for _, x := range []int{at, at + wide - 1} {
			got := s.Hit(x, line, e)
			if got.Field != pill.field || got.ID != "zzz-mine" {
				t.Errorf("column %d of the %s pill = %+v", x, pill.field, got)
			}
		}

		at += wide + rowPillGap
	}

	if got := flow.List(e.Flows); len(got) != len(flow.BuiltinNames())+1 {
		t.Fatalf("expected 5 flows listed, got %d", len(got))
	}
}

// flowRowOf is the row a flow's own line was drawn on: the one that names it and
// says where it came from, which is the row the buttons sit on.
func flowRowOf(t *testing.T, s State, name string, e Env) int {
	t.Helper()

	for i, row := range s.View(e.Frame.Body.H, e.Frame.Body.W, e) {
		if strings.Contains(ansi.Strip(row), name+"  (") {
			return i
		}
	}

	t.Fatalf("the list draws no row for %q", name)

	return 0
}

// TestTheListScrollsAndTheRailSaysSo. Eight flows with phases are taller
// than any terminal: fill cut the rest, so the reader could not see the last
// three and nothing on screen said they were there.
func TestTheListScrollsAndTheRailSaysSo(t *testing.T) {
	_, e := designer(t)
	e.Flows = flowsTestDir(t.TempDir())
	s := Open(FromBoard, e)

	lines := s.flowsListLines(e.Frame.Body.W, e)
	if len(lines) <= e.Frame.Body.H {
		t.Skip("this build's flows fit the window, so there is nothing to scroll")
	}

	rows := s.View(e.Frame.Body.H, e.Frame.Body.W, e)
	if len(rows) != e.Frame.Body.H {
		t.Fatalf("the list drew %d rows into a body of %d", len(rows), e.Frame.Body.H)
	}

	if !strings.Contains(strings.Join(rows, "\n"), cells.Thumb) {
		t.Error("a list taller than the window drew no rail")
	}

	// The wheel moves it, and the click that lands afterwards is the row
	// the reader is looking at rather than the one that used to be there.
	// Three rows is what one notch of the wheel moves, which the window
	// decides and hands down.
	const notch = 3

	s = s.Scroll(notch)
	if s.scroll != notch {
		t.Fatalf("the wheel left the list at %d", s.scroll)
	}

	start := s.flowsListStart(lines, max(e.Frame.Body.H-1, 0))

	var flowRow, want int

	for i := start; i < len(lines); i++ {
		if lines[i].at != noFlow {
			flowRow, want = i, lines[i].at
			break
		}
	}

	got := s.Hit(10, e.Frame.Body.Y+flowRow-start, e)
	if got.Kind != point.FlowItem || got.ID != s.listed[want].Name {
		t.Errorf("a click after scrolling landed on %+v, want %q", got, s.listed[want].Name)
	}
}

// TestTheFloorOfTheListAnswersNoClick. The ways out are drawn outside the
// page, so the last row of the body belongs to no flow — but the hit-test
// counted it as one more row of the list and answered the row underneath
// the window: the line that says what the keys do inspected a flow that
// was not on the screen, and on a body short enough that the create button
// was the row below the page, clicking it opened the designer.
func TestTheFloorOfTheListAnswersNoClick(t *testing.T) {
	for _, h := range []int{6, 8, 12, 14, 24} {
		e := world(t)

		frame, err := layout.Fit(100, h)
		if err != nil {
			continue
		}

		e.Frame = frame
		s := Open(FromBoard, e)

		body := e.Frame.Body.H

		rows := s.View(body, e.Frame.Body.W, e)
		floor := ansi.Strip(rows[len(rows)-1])

		if !strings.Contains(floor, "back") {
			t.Fatalf("at %d rows the last line is %q, want the ways out", body, floor)
		}

		if got := s.Hit(4, e.Frame.Body.Y+body-1, e); got.Kind != point.None {
			t.Errorf("at %d rows a click on the ways out answers %v %q %q",
				body, got.Kind, got.Field, got.ID)
		}
	}
}

// TestNoClickAnswersARowUnderTheWindow, at any height and anywhere down the
// list: what the page holds is what can be clicked.
func TestNoClickAnswersARowUnderTheWindow(t *testing.T) {
	e := world(t)

	frame, err := layout.Fit(100, 24)
	if err != nil {
		t.Skip("a 100x24 terminal does not fit this build's frame")
	}

	e.Frame = frame

	s := Open(FromBoard, e)
	body := e.Frame.Body.H
	lines := s.flowsListLines(e.Frame.Body.W, e)
	page := max(body-1, 0)

	if len(lines) <= page {
		t.Skip("this build's flows fit the window, so nothing is under it")
	}

	for _, scroll := range []int{0, 3, 6, len(lines)} {
		at := s
		at.scroll = scroll
		start := at.flowsListStart(lines, page)

		for line := range body {
			got := at.Hit(4, e.Frame.Body.Y+line, e)
			if got.Kind == point.None {
				continue
			}

			if line >= page {
				t.Errorf("scrolled to %d, row %d is under the page of %d and answers %v %q",
					start, line, page, got.Kind, got.Field)
			}
		}
	}
}
