package ui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/verb"
)

// TestTheWheelScrollsTheMapOverItsFiles. Every row of a map is a file you
// can click, and the wheel only scrolled over the pane's plain rows, so a
// long map barely moved under it.
func TestTheWheelScrollsTheMapOverItsFiles(t *testing.T) {
	m, _ := openDetail(t, "ACME-2662")

	cells := make([]verb.Cell, 0, 60)

	for i := range 60 {
		name := fmt.Sprintf("f%02d.go", i)
		cells = append(cells, verb.Cell{Name: name, Path: name, Changed: 1, Lines: 2})
	}

	m.shape.known = true
	m.shape.tree = verb.Cell{Changed: 60, Lines: 120, Cells: cells}
	m.tab = tabMap
	m = m.syncPanes()

	y := -1

	for row := range m.frame.Body.H {
		at := m.frame.Body.Y + row
		if m.hit(10, at).Kind == point.MapFile {
			y = at

			break
		}
	}

	if y < 0 {
		t.Fatal("no row of the map is a file under the pointer")
	}

	before := m.panes[tabMap].YOffset()
	after := m.wheel(tea.MouseWheelMsg{X: 10, Y: y, Button: tea.MouseWheelDown}.Mouse())

	if got := after.panes[tabMap].YOffset(); got <= before {
		t.Errorf("a notch over a file left the map at offset %d, want past %d", got, before)
	}
}
