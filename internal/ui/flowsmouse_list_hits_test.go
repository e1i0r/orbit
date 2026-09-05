package ui

// flowsmouse_coverage_test.go is wrapPromptText and nextOption's remaining
// branches, and hitFlows's list view — what a click resolves to before the
// builder is open. flowsmouse_builder_coverage_test.go is the builder's own
// half of hitFlows, which has a geometry all its own.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/flow"
)

func TestWrapPromptText(t *testing.T) {
	if got := wrapPromptText("", 10); got != nil {
		t.Errorf("wrapPromptText(\"\", 10) = %v, want nil", got)
	}

	if got := wrapPromptText("hello", 0); got != nil {
		t.Errorf("wrapPromptText with maxLen 0 = %v, want nil", got)
	}

	if got := wrapPromptText("   ", 10); got != nil {
		t.Errorf("wrapPromptText of only whitespace = %v, want nil", got)
	}

	if got := wrapPromptText("hello world", 100); len(got) != 1 || got[0] != "hello world" {
		t.Errorf("wrapPromptText that fits on one line = %v", got)
	}

	got := wrapPromptText("aaaaaaaaaa bbbbbbbbbb cccccccccc", 12)
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
	if got := nextOption(nil, "x", 1); got != "x" {
		t.Errorf("nextOption with no options = %q, want the current value unchanged", got)
	}

	if got := nextOption([]string{"a", "b", "c"}, "not-there", 1); got != "b" {
		t.Errorf("nextOption with an unmatched current = %q, want it to treat the start as index 0", got)
	}

	if got := nextOption([]string{"a", "b", "c"}, "a", -1); got != "c" {
		t.Errorf("nextOption wrapping backward past the start = %q, want c", got)
	}

	if got := nextOption([]string{"a", "b", "c"}, "c", 1); got != "a" {
		t.Errorf("nextOption wrapping forward past the end = %q, want a", got)
	}
}

func TestHitFlowsOutsideBody(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m = m.openFlows()
	if got := m.hitFlows(10, 0); got.Kind != TargetNone {
		t.Errorf("hitFlows outside the body = %+v, want the zero Target", got)
	}
}

func TestHitFlowsListCreateButton(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openFlows()
	y := m.frame.Body.Y + 4

	got := m.hitFlows(10, y)
	if got.Kind != TargetFlowItem || got.Field != "create" {
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
	m, _ := testModel(t, 100, 60)
	m = m.openFlows()
	base := m.frame.Body.Y

	drawn := m.flowsRows(m.frame.Body.H, m.frame.Body.W)

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

		got := m.hitFlows(10, base+line)
		if got.Kind != TargetFlowItem || got.ID != name {
			t.Errorf("hitFlows on the row of %q = %+v", name, got)
		}
	}

	// Past every listed flow, there is nothing to click. Counted from what
	// was drawn rather than from a line written here, for the reason above.
	if got := m.hitFlows(10, base+len(drawn)+1); got.Kind != TargetNone {
		t.Errorf("hitFlows past the last flow = %+v, want the zero Target", got)
	}
}

// TestHitFlowsListDeleteVsEdit is a reader's own flow, which offers details,
// edit, and delete buttons when selected.
func TestHitFlowsListDeleteVsEdit(t *testing.T) {
	dir := t.TempDir()
	flowJSON := `{"name":"zzz-mine","phases":[{"name":"implement","engine":"claude"}]}`
	writeFlowFile(t, dir, "zzz-mine", flowJSON)

	m, _ := testModel(t, 100, 50)
	m.opts.Flows = flowsTestDir(dir)
	m = m.openFlows()
	m.flows.sel = len(flow.BuiltinNames()) // the reader's own, after the builtins
	// The flow's own row, found by reading what was drawn: it is several
	// rows tall and how many depends on its phases, so a line counted here
	// goes wrong the day any flow before it changes.
	base := m.frame.Body.Y
	line := base + flowRowOf(t, m, "zzz-mine")

	// Clicking flow name opens details
	if got := m.hitFlows(10, line); got.Field != "details" || got.ID != "zzz-mine" {
		t.Errorf("hitFlows on flow name = %+v, want a details click", got)
	}
	// Clicking Edit button
	if got := m.hitFlows(44, line); got.Field != "edit" || got.ID != "zzz-mine" {
		t.Errorf("hitFlows on the edit pill = %+v, want an edit click", got)
	}
	// Clicking Delete button
	if got := m.hitFlows(56, line); got.Field != "delete" || got.ID != "zzz-mine" {
		t.Errorf("hitFlows on the delete pill = %+v, want a delete click", got)
	}

	if got := flow.List(m.opts.Flows); len(got) != len(flow.BuiltinNames())+1 {
		t.Fatalf("expected 5 flows listed, got %d", len(got))
	}
}

// flowRowOf is the row a flow's own line was drawn on: the one that names it and
// says where it came from, which is the row the buttons sit on.
func flowRowOf(t *testing.T, m Model, name string) int {
	t.Helper()

	for i, row := range m.flowsRows(m.frame.Body.H, m.frame.Body.W) {
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
	m, _ := testModel(t, 100, 24)
	m.opts.Flows = flowsTestDir(t.TempDir())
	m = m.openFlows()

	lines := m.flowsListLines(m.frame.Body.W)
	if len(lines) <= m.frame.Body.H {
		t.Skip("this build's flows fit the window, so there is nothing to scroll")
	}

	rows := m.flowsRows(m.frame.Body.H, m.frame.Body.W)
	if len(rows) != m.frame.Body.H {
		t.Fatalf("the list drew %d rows into a body of %d", len(rows), m.frame.Body.H)
	}

	if !strings.Contains(strings.Join(rows, "\n"), scrollThumb) {
		t.Error("a list taller than the window drew no rail")
	}

	// The wheel moves it, and the click that lands afterwards is the row
	// the reader is looking at rather than the one that used to be there.
	m = m.wheel(tea.Mouse{X: 10, Y: m.frame.Body.Y + 2, Button: tea.MouseWheelDown})
	if m.flows.scroll != wheelRows {
		t.Fatalf("the wheel left the list at %d", m.flows.scroll)
	}

	start := m.flowsListStart(lines, max(m.frame.Body.H-1, 0))

	var flowRow, want int

	for i := start; i < len(lines); i++ {
		if lines[i].at != noFlow {
			flowRow, want = i, lines[i].at
			break
		}
	}

	got := m.hitFlows(10, m.frame.Body.Y+flowRow-start)
	if got.Kind != TargetFlowItem || got.ID != m.flows.listed[want].Name {
		t.Errorf("a click after scrolling landed on %+v, want %q", got, m.flows.listed[want].Name)
	}
}
