package ui

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

func TestDiffSelectRenderAndMouseHit(t *testing.T) {
	files := []diffFile{
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
	res, _ := m.leftClick(Target{Kind: TargetDiffSelectToggle})

	mRes, ok := res.(Model)
	if !ok || !mRes.diffFilePicker {
		t.Fatal("expected diffFilePicker to be true after toggle")
	}

	// Click to select file 0
	res2, _ := mRes.leftClick(Target{Kind: TargetDiffFile, Pane: 0})

	mRes2, ok2 := res2.(Model)
	if !ok2 || mRes2.diffFilePicker {
		t.Fatal("expected diffFilePicker to close after file selection")
	}
}
