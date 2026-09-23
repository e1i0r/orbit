package flows

import (
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

// TestARowNotChosenHasNoButtons. Only the chosen row draws Details, Edit
// and Delete; the same columns on any other row are blank, and a click
// there opened the designer, or deleted, on a flow that showed no button.
// There it is a click on the row.
func TestARowNotChosenHasNoButtons(t *testing.T) {
	dir := t.TempDir()
	writeFlowFile(t, dir, "zzz-mine",
		`{"name":"zzz-mine","phases":[{"name":"implement","engine":"claude"}]}`)

	_, e := designer(t)
	e.Flows = flowsTestDir(dir)
	s := Open(FromBoard, e)
	s.sel = 0 // a built-in chosen, not the reader's own

	line := e.Frame.Body.Y + flowRowOf(t, s, "zzz-mine", e)

	var mine flow.Listed

	for _, d := range flow.List(e.Flows) {
		if d.Name == "zzz-mine" {
			mine = d
		}
	}

	at := rowPillsStart(mine, e)

	for _, pill := range rowPills(mine, e) {
		wide := lipgloss.Width(pill.drawn)

		for x := at; x < at+wide; x++ {
			if got := s.Hit(x, line, e); got.Field != "details" || got.ID != "zzz-mine" {
				t.Errorf("column %d, where the %s pill is on the chosen row, = %+v; want a click on the row",
					x, pill.field, got)
			}
		}

		at += wide + rowPillGap
	}
}
