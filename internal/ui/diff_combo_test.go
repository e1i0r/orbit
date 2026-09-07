package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/patch"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/words"
)

func TestDiffSelectRenderAndMouseHit(t *testing.T) {
	files := []patch.File{
		{Path: "internal/ui/badge.go", StartLine: 10, Added: 5, Deleted: 2, Status: "M"},
		{Path: "internal/ui/screen.go", StartLine: 35, Added: 12, Deleted: 0, Status: "M"},
	}
	p := words.For("en")

	// Closed state
	closed := renderDiffFileSelect(files, 0, 80, p, nil, false, 0)
	if !strings.Contains(closed, "badge.go") {
		t.Errorf("renderDiffFileSelect closed state missing badge.go, got %q", closed)
	}

	if !strings.Contains(closed, "[▾ select / f]") {
		t.Errorf("renderDiffFileSelect closed state missing dropdown hint, got %q", closed)
	}

	// Open dropdown state
	open := renderDiffFileSelect(files, 0, 80, p, nil, true, 1)
	if !strings.Contains(open, "badge.go") || !strings.Contains(open, "screen.go") {
		t.Errorf("renderDiffFileSelect open state missing files, got %q", open)
	}

	if !strings.Contains(open, "[▴ close / esc]") {
		t.Errorf("renderDiffFileSelect open state missing close hint, got %q", open)
	}

	// Model mouse interaction: toggle open and jump
	m, _ := testModel(t, 100, 30)
	m.tab = tabDiff
	m.diff = "diff --git a/internal/ui/badge.go b/internal/ui/badge.go\n+new"
	m.diffKnown = true

	// Click to open
	res, _ := m.leftClick(point.Target{Kind: point.DiffSelectToggle})

	mRes, ok := res.(Model)
	if !ok || !mRes.diffFilePicker {
		t.Fatal("expected diffFilePicker to be true after toggle")
	}

	// Click to select file 0
	res2, _ := mRes.leftClick(point.Target{Kind: point.DiffFile, Pane: 0})

	mRes2, ok2 := res2.(Model)
	if !ok2 || mRes2.diffFilePicker {
		t.Fatal("expected diffFilePicker to close after file selection")
	}
}

// TestTheFilePickerSaysThereIsMore. Nineteen files in a box that shows seven
// said nothing about the other twelve: the reader learned they were there by
// holding the arrow key down.
func TestTheFilePickerSaysThereIsMore(t *testing.T) {
	var files []patch.File
	for i := range 19 {
		files = append(files, patch.File{Path: fmt.Sprintf("internal/ui/file%d.go", i), Status: "MODIFIED"})
	}

	open := ansi.Strip(renderDiffFileSelect(files, 0, 100, words.For("en"), nil, true, 0))
	if !strings.Contains(open, cells.Thumb) {
		t.Errorf("a picker with more files than it shows drew no rail:\n%s", open)
	}

	// A list that fits has nothing to scroll, and draws no rail.
	short := ansi.Strip(renderDiffFileSelect(files[:3], 0, 100, words.For("en"), nil, true, 0))
	if strings.Contains(short, cells.Thumb) {
		t.Errorf("a picker showing everything drew a rail:\n%s", short)
	}
}

// TestTheWheelMovesThePickedFile. The picker is a list of nineteen in a box
// that shows seven, and a hand already on the mouse — it opened the box with
// it — was answered by the pane behind it instead.
func TestTheWheelMovesThePickedFile(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenDetail
	m.tab = tabDiff
	m.diffKnown = true

	var b strings.Builder
	for i := range 19 {
		fmt.Fprintf(&b, "diff --git a/internal/ui/file%d.go b/internal/ui/file%d.go\n+new\n", i, i)
	}

	m.diff = b.String()
	m.diffFilePicker = true

	down := m.wheel(tea.Mouse{X: 20, Y: 10, Button: tea.MouseWheelDown})
	if down.diffFileCursor != wheelRows {
		t.Errorf("a notch down moved the pick to %d, not %d", down.diffFileCursor, wheelRows)
	}

	if up := down.wheel(tea.Mouse{X: 20, Y: 10, Button: tea.MouseWheelUp}); up.diffFileCursor != 0 {
		t.Errorf("a notch back up left the pick at %d", up.diffFileCursor)
	}

	// The pane behind it is not scrolled while the box is open.
	if down.panes[tabDiff].YOffset() != m.panes[tabDiff].YOffset() {
		t.Error("the wheel scrolled the diff under the open picker")
	}
}
