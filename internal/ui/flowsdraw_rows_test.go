package ui

// flowsdraw_coverage_test.go is flowOriginString's whole switch, and the
// two flowsRows branches the fixture board never reaches on its own: a
// selected reader's-own flow (the delete pill) and a flow file that fails
// to resolve (the error line in its place).

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
)

func TestFlowOriginString(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	p := m.opts.Words
	cases := []struct {
		origin flow.Origin
		want   string
	}{
		{flow.OriginBuiltin, "built in"},
		{flow.OriginUser, "yours"},
		{flow.OriginShadow, "shadowing"},
		{flow.OriginUnknown, ""},
	}
	for _, c := range cases {
		got := flowOriginString(p, c.origin)
		if c.want == "" {
			if got != "" {
				t.Errorf("flowOriginString(%v) = %q, want empty", c.origin, got)
			}
			continue
		}
		if !strings.Contains(got, c.want) {
			t.Errorf("flowOriginString(%v) = %q, want it to mention %q", c.origin, got, c.want)
		}
	}
}

func TestFlowsRowsSelectedUserFlow(t *testing.T) {
	dir := t.TempDir()
	writeFlowFile(t, dir, "zzz-mine", `{"name":"zzz-mine","phases":[{"name":"implement","engine":"claude"}]}`)

	m, _ := testModel(t, 100, 50)
	m.opts.Flows = flowsTestDir(dir)
	m = m.openFlows()
	m.flows.sel = 4 // careful, quick, task, tdd-fuzz-pr, zzz-mine

	rows := m.flowsRows(m.frame.Body.H, m.frame.Body.W)
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "Borrar") && !strings.Contains(joined, "Delete") {
		t.Errorf("expected a delete pill on a selected reader's-own flow")
	}
	if !strings.Contains(joined, "Editar") && !strings.Contains(joined, "Edit") {
		t.Errorf("expected an edit pill on the selected flow")
	}
}

func TestFlowsRowsResolveError(t *testing.T) {
	dir := t.TempDir()
	// A file whose internal name disagrees with its filename fails
	// flow.Resolve's own naming check, which is the one error flowsRows
	// draws in place of a flow's phases.
	writeFlowFile(t, dir, "mismatched", `{"name":"not-mismatched","phases":[{"name":"implement","engine":"claude"}]}`)

	m, _ := testModel(t, 100, 30)
	m.opts.Flows = flowsTestDir(dir)
	m = m.openFlows()

	rows := m.flowsRows(m.frame.Body.H, m.frame.Body.W)
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "mismatched") {
		t.Errorf("expected the broken flow's name in the listing")
	}
}
