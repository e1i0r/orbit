package patch

// A diff read: which files it touches, how much of each, and where each
// hunk starts. The line numbers are what the pane scrolls to, so a file
// whose bounds are wrong is a click that lands somewhere else.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/words"
)

// aDiff is one of each kind of file: one modified, one added, one deleted.
func aDiff() []string {
	return strings.Split(`diff --git a/internal/ui/rows.go b/internal/ui/rows.go
index 1111111..2222222 100644
--- a/internal/ui/rows.go
+++ b/internal/ui/rows.go
@@ -10,7 +10,7 @@ func rows() {
-	old line
+	new line
+	another new line
diff --git a/internal/ui/new.go b/internal/ui/new.go
new file mode 100644
index 0000000..3333333
--- /dev/null
+++ b/internal/ui/new.go
@@ -0,0 +1,2 @@
+package ui
+
diff --git a/internal/ui/gone.go b/internal/ui/gone.go
deleted file mode 100644
index 4444444..0000000
--- a/internal/ui/gone.go
+++ /dev/null
@@ -1,2 +0,0 @@
-package ui
-`, "\n")
}

// TestEachFileIsReadWithItsOwnBoundsAndCount.
func TestEachFileIsReadWithItsOwnBoundsAndCount(t *testing.T) {
	lines := aDiff()

	files := Files(lines)
	if len(files) != 3 {
		t.Fatalf("the diff was read as %d files, want three", len(files))
	}

	for i, c := range []struct {
		path    string
		status  string
		added   int
		deleted int
		hunks   int
	}{
		{"internal/ui/rows.go", "MOD", 2, 1, 1},
		{"internal/ui/new.go", "NEW", 2, 0, 1},
		{"internal/ui/gone.go", "DEL", 0, 2, 1},
	} {
		f := files[i]
		if f.Path != c.path || f.Status != c.status {
			t.Errorf("file %d is %q %q, want %q %q", i, f.Path, f.Status, c.path, c.status)
		}

		if f.Added != c.added || f.Deleted != c.deleted {
			t.Errorf("%s counted +%d -%d, want +%d -%d", c.path, f.Added, f.Deleted, c.added, c.deleted)
		}

		if len(f.Hunks) != c.hunks {
			t.Errorf("%s has %d hunks, want %d", c.path, len(f.Hunks), c.hunks)
		}
	}

	// The bounds are what the pane scrolls to: each file starts on its own
	// header and ends where the next one begins.
	if files[0].StartLine != 0 {
		t.Errorf("the first file starts at line %d", files[0].StartLine)
	}

	for i := range files[:len(files)-1] {
		if files[i].EndLine+1 != files[i+1].StartLine {
			t.Errorf("file %d ends at %d and file %d starts at %d, leaving a gap",
				i, files[i].EndLine, i+1, files[i+1].StartLine)
		}
	}

	if last := files[len(files)-1]; last.EndLine != len(lines)-1 {
		t.Errorf("the last file ends at %d, want the last line, %d", last.EndLine, len(lines)-1)
	}
}

// TestLinesBeforeAnyFileBelongToNoFile. A diff that opens with a summary
// would otherwise have its own header counted as somebody's change.
func TestLinesBeforeAnyFileBelongToNoFile(t *testing.T) {
	got := Files([]string{"commit 1234", "+not a change", "-nor this"})
	if len(got) != 0 {
		t.Errorf("lines before any file header were read as %+v", got)
	}
}

// TestTheTotalsAreTheFilesAddedUp.
func TestTheTotalsAreTheFilesAddedUp(t *testing.T) {
	added, deleted := Stats(Files(aDiff()))
	if added != 4 || deleted != 3 {
		t.Errorf("the diff totals +%d -%d, want +4 -3", added, deleted)
	}

	if a, d := Stats(nil); a != 0 || d != 0 {
		t.Errorf("no files total +%d -%d", a, d)
	}
}

// TestEachStatusIsSaidInItsOwnWordAndColour, so a deletion is never read as
// a modification at a glance.
func TestEachStatusIsSaidInItsOwnWordAndColour(t *testing.T) {
	seen := map[string]bool{}

	for _, c := range []struct{ status, want string }{
		{"NEW", "NEW"},
		{"DEL", "DELETED"},
		{"MOD", "MODIFIED"},
		{"anything else", "MODIFIED"},
	} {
		got := Badge(c.status)
		if !strings.Contains(ansi.Strip(got), c.want) {
			t.Errorf("Badge(%q) reads %q, want it to say %q", c.status, ansi.Strip(got), c.want)
		}

		seen[got] = true
	}

	// Three words, three paints: NEW, DELETED and MODIFIED are not drawn
	// the same.
	if len(seen) != 3 {
		t.Errorf("the badges are drawn %d different ways, want three", len(seen))
	}
}

// TestAFileIsIconedByWhatItIs.
func TestAFileIsIconedByWhatItIs(t *testing.T) {
	for _, c := range []struct{ path, want string }{
		{"internal/ui/rows.go", "🔷"},
		{"go.mod", "📄"},
		{"orbit.json", "⚙️ "},
		{"config.yaml", "⚙️ "},
		{"README.md", "📝"},
		{"notes.txt", "📝"},
		{"app.tsx", "🟨"},
		{"main.py", "🐍"},
		{"install.sh", "🐚"},
	} {
		if got := Icon(c.path); got != c.want {
			t.Errorf("Icon(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

// TestAHunkHeaderKeepsTheFunctionItIsIn, which is the only thing on that
// line that says where in the file the reader is.
func TestAHunkHeaderKeepsTheFunctionItIsIn(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"@@ -10,7 +10,9 @@ func rows() {", " @@ -10,7 +10,9 @@ func rows() {"},
		{"@@ -1 +1 @@", " @@ -1 +1 @@"},
		{"not a hunk header", "not a hunk header"},
	} {
		if got := HunkHeader(c.in); got != c.want {
			t.Errorf("HunkHeader(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestAFileWithNothingWrittenAboutItStillSaysWhatItIs. The record says why a
// file changed when it can; when it cannot, the file's own name is the only
// honest thing left to say.
func TestAFileWithNothingWrittenAboutItStillSaysWhatItIs(t *testing.T) {
	p := words.For("en")

	for _, c := range []struct {
		file File
		want string
	}{
		{File{Path: "internal/ui/rows_test.go", Status: "MOD"}, "testing"},
		{File{Path: "internal/ui/new.go", Status: "NEW"}, "new"},
		{File{Path: "internal/ui/gone.go", Status: "DEL"}, "removed"},
		{File{Path: "internal/ui/rows.go", Status: "MOD"}, "updated"},
	} {
		if got := fallback(c.file, p); !strings.Contains(strings.ToLower(got), c.want) {
			t.Errorf("%s (%s) reads %q, want it to mention %q", c.file.Path, c.file.Status, got, c.want)
		}
	}
}
