package panes

import (
	"path"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/verb"
)

// TestEveryFileOnTheMapIsFoundOnTheRowItIsDrawnOn. The click that opens a
// file's diff is only as good as this: a row off by one opens the file
// above.
func TestEveryFileOnTheMapIsFoundOnTheRowItIsDrawnOn(t *testing.T) {
	e := world(t, nil)
	e.Shape = shaped(verb.Cell{
		Changed: 3, Lines: 30,
		Cells: []verb.Cell{
			{Name: "webhook", Path: "webhook", Changed: 2, Lines: 20, Cells: []verb.Cell{
				{Name: "pay.go", Path: "webhook/pay.go", Changed: 1, Lines: 20},
				{Name: "idle.go", Path: "webhook/idle.go"},
				{Name: "retry.go", Path: "webhook/retry.go", Changed: 1},
			}},
			{Name: "main.go", Path: "main.go", Changed: 1, Lines: 10},
		},
	}, true, false, false, "")

	rows := Map(e)
	files := MapFiles(e)

	if len(files) != 3 {
		t.Fatalf("the map offers %d files, want the 3 the task touched: %v", len(files), files)
	}

	for at, file := range files {
		if at >= len(rows) {
			t.Fatalf("%s is said to be on row %d of %d", file, at, len(rows))
		}

		if drawn := ansi.Strip(rows[at]); !strings.Contains(drawn, path.Base(file)) {
			t.Errorf("row %d is %q, want the row of %s", at, drawn, file)
		}
	}

	if _, on := files[mapHeadRows]; on {
		t.Error("the webhook directory is offered as a file")
	}
}
