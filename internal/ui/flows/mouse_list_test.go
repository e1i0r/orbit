package flows

// flowsmouse_coverage_test.go is wrapPromptText and nextOption's remaining
// branches, and hitFlows's list view — what a click resolves to before the
// builder is open. flowsmouse_builder_coverage_test.go is the builder's own
// half of hitFlows, which has a geometry all its own.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
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
	y := e.Frame.Body.Y + 4

	got := s.Hit(10, y, e)
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
	// Clicking Edit button
	if got := s.Hit(44, line, e); got.Field != "edit" || got.ID != "zzz-mine" {
		t.Errorf("hitFlows on the edit pill = %+v, want an edit click", got)
	}
	// Clicking Delete button
	if got := s.Hit(56, line, e); got.Field != "delete" || got.ID != "zzz-mine" {
		t.Errorf("hitFlows on the delete pill = %+v, want a delete click", got)
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
