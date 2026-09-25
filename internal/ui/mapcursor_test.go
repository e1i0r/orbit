package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/verb"
)

// TestTheMapIsWalkedByTheKeyboard. A file on the map opened its diff by a
// click and by nothing else. ↓ and ↑ walk the files, round the ends, the
// caret is drawn on the one they stand on, and ↵ opens its diff.
func TestTheMapIsWalkedByTheKeyboard(t *testing.T) {
	m, _ := testModel(t, 100, 40)
	m.screen, m.detail = screenDetail, "ACME-2662"
	m.diff = "diff --git a/a.go b/a.go\n--- a/a.go\n+++ b/a.go\n@@ -1 +1 @@\n-a\n+b\n" +
		"diff --git a/b.go b/b.go\n--- a/b.go\n+++ b/b.go\n@@ -1 +1 @@\n-c\n+d\n"
	m.diffKnown = true
	m.shape.known = true
	m.shape.tree = verb.Cell{Changed: 2, Lines: 4, Cells: []verb.Cell{
		{Name: "a.go", Path: "a.go", Changed: 1, Lines: 2},
		{Name: "b.go", Path: "b.go", Changed: 1, Lines: 2},
	}}
	m.tab = tabMap
	m = m.syncPanes()

	press := func(k string) Model {
		t.Helper()

		next, _ := m.detailKey(keystroke(k))

		return asModel(t, next)
	}

	for _, step := range []struct{ key, want string }{
		{"down", "a.go"}, {"down", "b.go"}, {"down", "a.go"}, {"up", "b.go"},
	} {
		if m = press(step.key); m.mapAt != step.want {
			t.Fatalf("%s left the cursor on %q, want %s", step.key, m.mapAt, step.want)
		}
	}

	drawn := ansi.Strip(strings.Join(m.mapRows(), "\n"))
	if !strings.Contains(drawn, treeCaret+"└─ b.go") {
		t.Errorf("the caret is not on b.go:\n%s", drawn)
	}

	if m = press("enter"); m.tab != tabDiff {
		t.Errorf("↵ on the map left tab %v, want the diff", m.tab)
	}
}
