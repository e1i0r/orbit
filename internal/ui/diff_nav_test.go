package ui

import (
	"strings"
	"testing"
)

const sampleDiff = `diff --git a/internal/ui/badge.go b/internal/ui/badge.go
new file mode 100644
--- /dev/null
+++ b/internal/ui/badge.go
@@ -0,0 +1,20 @@
+package ui
+
+func FormatLatency() {}
diff --git a/internal/ui/badge_test.go b/internal/ui/badge_test.go
new file mode 100644
--- /dev/null
+++ b/internal/ui/badge_test.go
@@ -0,0 +1,45 @@
+package ui_test
+
+func TestFormatLatency() {}
`

func TestParseDiffFiles(t *testing.T) {
	lines := parseDiffFiles(strings.Split(sampleDiff, "\n"))
	if len(lines) != 2 {
		t.Fatalf("expected 2 files parsed, got %d", len(lines))
	}
	if lines[0].Path != "internal/ui/badge.go" {
		t.Errorf("file 0 path = %q, want internal/ui/badge.go", lines[0].Path)
	}
	if lines[0].Status != "NEW" {
		t.Errorf("file 0 status = %q, want NEW", lines[0].Status)
	}
	if lines[1].Path != "internal/ui/badge_test.go" {
		t.Errorf("file 1 path = %q, want internal/ui/badge_test.go", lines[1].Path)
	}

	add, del := diffStats(lines)
	if add == 0 {
		t.Errorf("expected added lines > 0, got %d", add)
	}
	if del != 0 {
		t.Errorf("expected deleted lines == 0, got %d", del)
	}
}

func TestDiffNavJumps(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.tab = tabDiff
	m.diff = sampleDiff

	// Jump next file
	m = m.jumpNextDiffFile()
	offset1 := m.panes[tabDiff].YOffset()

	// Jump next file again
	m = m.jumpNextDiffFile()
	offset2 := m.panes[tabDiff].YOffset()

	// Jump prev file
	m = m.jumpPrevDiffFile()
	offset3 := m.panes[tabDiff].YOffset()

	if offset2 <= offset1 && offset3 != offset1 {
		t.Errorf("offsets jump: %d -> %d -> %d", offset1, offset2, offset3)
	}
}
