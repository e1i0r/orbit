package ui

import (
	"testing"
)

func TestDiffCollapseAndExpand(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.tab = tabDiff
	m.diff = `diff --git a/internal/ui/badge.go b/internal/ui/badge.go
new file mode 100644
--- /dev/null
+++ b/internal/ui/badge.go
@@ -0,0 +1,20 @@
+package ui
+func FormatLatency() {}
diff --git a/internal/ui/badge_test.go b/internal/ui/badge_test.go
new file mode 100644
--- /dev/null
+++ b/internal/ui/badge_test.go
@@ -0,0 +1,15 @@
+package ui_test
+func TestFormatLatency() {}
`
	m.diffKnown = true

	// Toggle collapse current file
	m = m.toggleCollapseCurrentFile()
	if !m.collapsedFiles["internal/ui/badge.go"] {
		t.Errorf("expected badge.go to be collapsed")
	}

	// Toggle collapse all
	m = m.toggleCollapseAll()
	if !m.collapsedFiles["internal/ui/badge_test.go"] {
		t.Errorf("expected badge_test.go to be collapsed after collapse all")
	}

	// Toggle expand all
	m = m.toggleCollapseAll()
	if m.collapsedFiles["internal/ui/badge.go"] || m.collapsedFiles["internal/ui/badge_test.go"] {
		t.Errorf("expected all files to be expanded")
	}
}

func TestDiffFilePicker(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.tab = tabDiff
	m.diff = `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1 +1 @@
-a
+b
diff --git a/b.go b/b.go
--- a/b.go
+++ b/b.go
@@ -1 +1 @@
-c
+d
`
	m.diffKnown = true

	m = m.openDiffFilePicker()
	if !m.diffFilePicker {
		t.Fatal("expected diffFilePicker to be true")
	}

	// Down arrow
	res, _ := m.handleDiffFilePickerKey(keystroke("down"))

	var ok bool
	if m, ok = res.(Model); !ok {
		t.Fatal("expected Model from handleDiffFilePickerKey")
	}

	if m.diffFileCursor != 1 {
		t.Errorf("expected cursor=1, got %d", m.diffFileCursor)
	}

	// Space to collapse
	res, _ = m.handleDiffFilePickerKey(keystroke("space"))
	if m, ok = res.(Model); !ok {
		t.Fatal("expected Model from handleDiffFilePickerKey")
	}

	if !m.collapsedFiles["b.go"] {
		t.Errorf("expected b.go to be collapsed")
	}

	// Enter to jump and close
	res, _ = m.handleDiffFilePickerKey(keystroke("enter"))
	if m, ok = res.(Model); !ok {
		t.Fatal("expected Model from handleDiffFilePickerKey")
	}

	if m.diffFilePicker {
		t.Errorf("expected diffFilePicker to be closed")
	}
}
