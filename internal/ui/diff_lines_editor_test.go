package ui

import (
	"errors"
	"strings"
	"testing"
)

func TestDiffLinesAllStates(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	// 1. Pending state (!diffKnown)
	m.diffKnown = false
	lines := m.diffLines()
	if len(lines) == 0 || !strings.Contains(lines[0], "reading") {
		t.Errorf("expected pending message, got %v", lines)
	}

	// 2. Timed out error
	m.diffKnown = true
	m.diffErr = errGitTimedOut
	lines = m.diffLines()
	if len(lines) == 0 || !strings.Contains(lines[0], "in time") {
		t.Errorf("expected timed out message, got %v", lines)
	}

	// 3. Other error
	m.diffErr = errors.New("custom git failure")
	lines = m.diffLines()
	if len(lines) == 0 || !strings.Contains(lines[0], "custom git failure") {
		t.Errorf("expected custom error message, got %v", lines)
	}

	// 4. Empty diff (no changes)
	m.diffErr = nil
	m.diff = ""
	lines = m.diffLines()
	if len(lines) == 0 || !strings.Contains(lines[0], "no changes") {
		t.Errorf("expected no changes message, got %v", lines)
	}

	// 5. Rich diff with additions, deletions, headers
	sampleDiff := `diff --git a/main.go b/main.go
index 1234567..89abcdef 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
-import "fmt"
+import "os"
+import "strings"
`
	m.diff = sampleDiff
	lines = m.diffLines()
	if len(lines) < 5 {
		t.Errorf("expected formatted diff lines, got %d lines", len(lines))
	}
}

func TestFileHeaderEveryForm(t *testing.T) {
	tests := []struct {
		line     string
		wantName string
		wantOK   bool
	}{
		{"+++ b/retry.go", "retry.go", true},
		{"+++ retry.go", "retry.go", true},
		{"+++ /dev/null", "", false},
		{"something else entirely", "", false},
	}
	for _, tt := range tests {
		name, ok := fileHeader(tt.line)
		if name != tt.wantName || ok != tt.wantOK {
			t.Errorf("fileHeader(%q) = (%q, %v), want (%q, %v)", tt.line, name, ok, tt.wantName, tt.wantOK)
		}
	}
}

func TestFileBelowEveryOutcome(t *testing.T) {
	// A hunk met before any file header: nothing to introduce.
	if _, _, ok := fileBelow([]string{"@@ -1,2 +1,2 @@", "+++ b/x.go"}, 0); ok {
		t.Error("fileBelow with a hunk first found a file, want none")
	}
	// A plain header: the file it introduces.
	name, line, ok := fileBelow([]string{"diff --git a/x.go b/x.go", "+++ b/x.go"}, 0)
	if !ok || name != "x.go" || line != 1 {
		t.Errorf("fileBelow with a header = (%q, %d, %v), want (x.go, 1, true)", name, line, ok)
	}
	// The file being deleted: refused outright.
	if _, _, ok := fileBelow([]string{"--- a/x.go", "+++ /dev/null"}, 0); ok {
		t.Error("fileBelow into a deleted file's furniture found one, want none")
	}
	// Nothing at all past the cursor.
	if _, _, ok := fileBelow([]string{"diff --git a/x.go b/x.go"}, 0); ok {
		t.Error("fileBelow with nothing ahead found a file, want none")
	}
}

// TestEditorForEveryRefusal walks editorFor's three refusals and its one
// success, all through the same fixture diff at the cursor's default
// position — the top of the pane, which is where fileAt's furniture-first
// walk lands on the file the first hunk belongs to.
func TestEditorForEveryRefusal(t *testing.T) {
	m := openOn(t, "ACME-2662")
	m.diff, m.diffKnown = fixtureDiff, true
	m = m.syncPanes()

	// 1. No $EDITOR and no $VISUAL.
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	if _, err := m.editorFor(); err == nil {
		t.Error("editorFor with no editor set returned no error")
	}

	// 2. An editor is set, but the cursor's line names no file at all.
	t.Setenv("EDITOR", "vim")
	deleted := m
	deleted.diff = "diff --git a/gone.go b/gone.go\n--- a/gone.go\n+++ /dev/null\n@@ -1,3 +0,0 @@\n-package gone\n"
	deleted = deleted.syncPanes()
	if _, err := deleted.editorFor(); err == nil {
		t.Error("editorFor on a deleted file's furniture returned no error")
	}

	// 3. A file is found, but the task has no worktree to open it in.
	noTree := m
	noTree.worktree = ""
	if _, err := noTree.editorFor(); err == nil {
		t.Error("editorFor with no worktree returned no error")
	}

	// 4. Success: the file at the top of the pane, in the task's worktree.
	m.worktree = "/w/ACME-2662"
	cmd, err := m.editorFor()
	if err != nil {
		t.Fatalf("editorFor: %v", err)
	}
	if cmd.Dir != "/w/ACME-2662" {
		t.Errorf("editorFor built a command in %q, want the worktree", cmd.Dir)
	}
}

func TestEditRefusesOffTheDiffTab(t *testing.T) {
	m := openOn(t, "ACME-2662")
	m.tab = tabOverview
	next, cmd := m.edit()
	got := asModel(t, next)
	if cmd != nil {
		t.Error("edit() off the diff tab produced a command")
	}
	wantBand(t, got, "only the diff tab")

	m.tab, m.diff, m.diffKnown, m.worktree = tabDiff, fixtureDiff, true, "/w/ACME-2662"
	m = m.syncPanes()
	t.Setenv("EDITOR", "vim")
	t.Setenv("VISUAL", "")
	next, cmd = m.edit()
	_ = asModel(t, next)
	if cmd == nil {
		t.Error("edit() on the diff tab with an editor available produced no command")
	}
}
