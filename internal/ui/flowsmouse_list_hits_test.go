package ui

// flowsmouse_coverage_test.go is wrapPromptText and nextOption's remaining
// branches, and hitFlows's list view — what a click resolves to before the
// builder is open. flowsmouse_builder_coverage_test.go is the builder's own
// half of hitFlows, which has a geometry all its own.

import (
	"testing"

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

// TestHitFlowsListEntries walks the descriptor list the way flowsRows draws
// it — careful (3 phases), quick (1), task (2) — and checks each header
// line resolves to an edit click, since every one of them is built in.
func TestHitFlowsListEntries(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openFlows()
	base := m.frame.Body.Y

	cases := []struct {
		line int
		name string
	}{
		{6, "careful"},
		{12, "quick"},
		{16, "task"},
		{21, "tdd-fuzz-pr"},
	}
	for _, c := range cases {
		got := m.hitFlows(10, base+c.line)
		if got.Kind != TargetFlowItem || got.Field != "edit" || got.ID != c.name {
			t.Errorf("hitFlows at line %d = %+v, want an edit click on %q", c.line, got, c.name)
		}
	}

	// Past every listed flow, there is nothing to click.
	if got := m.hitFlows(10, base+30); got.Kind != TargetNone {
		t.Errorf("hitFlows past the last flow = %+v, want the zero Target", got)
	}
}

// TestHitFlowsListDeleteVsEdit is a reader's own flow, which offers delete
// to the right of its name and edit everywhere else on the same line.
func TestHitFlowsListDeleteVsEdit(t *testing.T) {
	dir := t.TempDir()
	writeFlowFile(t, dir, "zzz-mine", `{"name":"zzz-mine","phases":[{"name":"implement","engine":"claude"}]}`)

	m, _ := testModel(t, 100, 50)
	m.opts.Flows = flowsTestDir(dir)
	m = m.openFlows()
	base := m.frame.Body.Y
	// careful(6..10) quick(12..14) task(16..19) tdd-fuzz-pr(21..25) zzz-mine(27..28)
	line := base + 27

	if got := m.hitFlows(10, line); got.Field != "edit" || got.ID != "zzz-mine" {
		t.Errorf("hitFlows left of the delete pill = %+v, want an edit click", got)
	}
	if got := m.hitFlows(40, line); got.Field != "delete" || got.ID != "zzz-mine" {
		t.Errorf("hitFlows on the delete pill = %+v, want a delete click", got)
	}
	if got := flow.List(m.opts.Flows); len(got) != 5 {
		t.Fatalf("expected 5 flows listed, got %d", len(got))
	}
}
