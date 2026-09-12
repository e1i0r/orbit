package panes

// The map pane: the repository as a tree, with what the task changed
// marked on it.
//
// Every state of the reading draws something of its own: asking, nothing
// read yet, a checkout gone, a failure, nothing changed, and the tree
// itself with its weights.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
)

// shaped is a Shape over one tree, in the state named.
func shaped(tree verb.Cell, read, asking, missing bool, failed string) Shape {
	return Shape{Tree: tree, Read: read, Asking: asking, Missing: missing, Failed: failed}
}

// TestMapReadsEveryStateOfItsReading.
func TestMapReadsEveryStateOfItsReading(t *testing.T) {
	e := world(t, nil)

	for name, shape := range map[string]Shape{
		"reading":     shaped(verb.Cell{}, false, true, false, ""),
		"not yet":     shaped(verb.Cell{}, false, false, false, ""),
		"gone":        shaped(verb.Cell{}, true, false, true, ""),
		"failed":      shaped(verb.Cell{}, true, false, false, "git would not answer"),
		"unchanged":   shaped(verb.Cell{}, true, false, false, ""),
		"nothing yet": {Read: true},
	} {
		e.Shape = shape

		if rows := Map(e); len(rows) == 0 {
			t.Errorf("the map in state %q drew nothing", name)
		}
	}
}

// TestMapDrawsWeightByWeight. Directories above files, the touched branch
// first, and every row carrying what it weighs.
func TestMapDrawsWeightByWeight(t *testing.T) {
	e := world(t, nil)
	e.Shape = shaped(verb.Cell{
		Name: "", Path: "", Changed: 3, Lines: 30,
		Cells: []verb.Cell{
			{Name: "webhook", Path: "webhook", Changed: 2, Lines: 20, Cells: []verb.Cell{
				{Name: "pay.go", Path: "webhook/pay.go", Changed: 1, Lines: 20},
				{Name: "retry.go", Path: "webhook/retry.go", Changed: 1, Lines: 0},
			}},
			{Name: "main.go", Path: "main.go", Changed: 1, Lines: 10},
		},
	}, true, false, false, "")

	joined := strings.Join(Map(e), "\n")

	for _, want := range []string{"webhook", "pay.go", "main.go", "3 files"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the map does not carry %q:\n%s", want, joined)
		}
	}
}
