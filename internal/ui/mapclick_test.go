package ui

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/patch"
	"github.com/e1i0r/orbit/internal/verb"
)

// TestAFileOnTheMapOpensItsDiff. Elio, on the map: clicking a file should
// take him to its diff, to see what was done there.
func TestAFileOnTheMapOpensItsDiff(t *testing.T) {
	m, _ := testModel(t, 100, 40)
	m.diff = "diff --git a/a.go b/a.go\n--- a/a.go\n+++ b/a.go\n@@ -1 +1 @@\n-a\n+b\n" +
		"diff --git a/b.go b/b.go\n--- a/b.go\n+++ b/b.go\n@@ -1 +1 @@\n-c\n+d\n" +
		strings.Repeat(" context\n", 60)
	m.diffKnown = true
	m.shape.known = true
	m.shape.tree = verb.Cell{Changed: 2, Lines: 4, Cells: []verb.Cell{
		{Name: "a.go", Path: "a.go", Changed: 1, Lines: 2},
		{Name: "b.go", Path: "b.go", Changed: 1, Lines: 2},
	}}
	m.tab = tabMap
	m = m.syncPanes()

	row := -1

	for r := range 10 {
		if file, on := m.mapFileAt(r); on && file == "b.go" {
			row = r
		}
	}

	if row < 0 {
		t.Fatal("no row of the map is b.go")
	}

	m = m.showDiffOf("b.go")
	if m.tab != tabDiff {
		t.Fatalf("the click left tab %v, want the diff", m.tab)
	}

	var want int

	for _, f := range patch.Files(strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")) {
		if f.Path == "b.go" {
			want = f.StartLine
		}
	}

	if got := m.panes[tabDiff].YOffset(); got != want {
		t.Errorf("the diff opened at line %d, want b.go's first line, %d", got, want)
	}
}
