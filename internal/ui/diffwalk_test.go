package ui

// diffwalk_test.go is the arithmetic behind o: which file a line of the diff
// belongs to, and which line of that file it lands on.
//
// It is a file of its own because that question is not a question about a
// frame — nothing here renders anything — and because the fixtures it needs
// are diffs of shapes no other test in this package wants: two files in one
// diff, a removed line that looks like a header, a file git is deleting.
//
// Getting this wrong opens the right file at the wrong line, or a different
// file altogether, which is worse than not opening anything at all: a reader
// believes what the editor shows them.

import (
	"strings"
	"testing"
)

// commentDiff is the fixture round 1 did not have and shipped green without.
//
// A removed line is rendered with a minus in front of it, so a deleted line
// whose own text begins `-- ` — SQL, Lua, Haskell, Ada; `-- the id is
// assigned by the gateway` is about as ordinary a removed line as there is —
// arrives as `--- the id is assigned by the gateway`, four characters that a
// file's old-side header also starts with. A walk that decided from the
// prefix alone read it as the start of the next file and answered webhook.go
// for it and for every line under it in the hunk.
const commentDiff = `diff --git a/schema.sql b/schema.sql
index 1a2b3c4..5d6e7f8 100644
--- a/schema.sql
+++ b/schema.sql
@@ -1,4 +1,4 @@
 create table payments (
--- the id is assigned by the gateway
+  id int,
   name text
 );
diff --git a/webhook.go b/webhook.go
index 9c1a2f0..1d4e6b3 100644
--- a/webhook.go
+++ b/webhook.go
@@ -1,3 +1,3 @@
 package webhook
-func old() {}
+func retry() {}
`

// deletedFileDiff is a file git is removing, with another file after it.
//
// Its new side is /dev/null, so there is no file in the worktree to open and
// no line of it to open at, and every one of its lines — the furniture and
// the removed lines under it — used to answer with whichever file came next
// in the diff, at line 1. The comment line in the middle of it is there for
// the same reason it is in commentDiff, and because a deleted SQL file is
// where the two shapes meet.
const deletedFileDiff = `diff --git a/legacy.sql b/legacy.sql
deleted file mode 100644
index 5d6e7f8..0000000
--- a/legacy.sql
+++ /dev/null
@@ -1,3 +0,0 @@
-create table old_payments (
--- the gateway stopped sending these
-);
diff --git a/webhook.go b/webhook.go
index 9c1a2f0..1d4e6b3 100644
--- a/webhook.go
+++ b/webhook.go
@@ -1,3 +1,3 @@
 package webhook
-func old() {}
+func retry() {}
`

// linesOf is a fixture diff as fileAt takes it: the lines, without the
// trailing empty one the final newline would otherwise make.
func linesOf(diff string) []string {
	return strings.Split(strings.TrimSuffix(diff, "\n"), "\n")
}

// TestTheDiffKnowsWhichFileALineBelongsTo walks the pair of functions o is
// built from. They are the only arithmetic in this screen, and getting them
// wrong opens the right file at the wrong line — which is worse than not
// opening it, because a reader believes what the editor shows them.
func TestTheDiffKnowsWhichFileALineBelongsTo(t *testing.T) {
	lines := strings.Split(strings.TrimSuffix(fixtureDiff, "\n"), "\n")
	cases := []struct {
		name string
		at   int
		file string
		line int
		ok   bool
	}{
		{"the first line of the hunk is the hunk's own start", 5, "retry.go", 28, true},
		{"a context line counts towards the new file", 6, "retry.go", 29, true},
		{"an added line is where it was added", 9, "retry.go", 32, true},
		{"the furniture above the first hunk is that file at its top", 1, "retry.go", 1, true},
		{"the line the pane opens on is the first file below it", 0, "retry.go", 1, true},
		{"a row past the end of the diff is not a file", 99, "", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file, line, ok := fileAt(lines, c.at)
			if ok != c.ok || file != c.file || line != c.line {
				t.Errorf("fileAt(%d) = %q, %d, %v; want %q, %d, %v", c.at, file, line, ok, c.file, c.line, c.ok)
			}
		})
	}
	// The removal rule needs a diff that removes something, and the fixture
	// above only adds. A line that was taken out is not in the new file at
	// all, so the line under the one that follows it has not moved.
	t.Run("a removed line does not advance the new file's count", func(t *testing.T) {
		wide := strings.Split(strings.TrimSuffix(wideDiff(), "\n"), "\n")
		if file, line, ok := fileAt(wide, 5); file != "bundle.js" || line != 1 || !ok {
			t.Errorf("fileAt(5) = %q, %d, %v; want %q, 1, true", file, line, ok, "bundle.js")
		}
	})
	// The bug I1 fixes needs two files in one diff, which every fixture
	// above is not. Walking up from a line in the furniture that introduces
	// the second file, fileAt used to meet that furniture before it met the
	// first file's own hunk header, and answered with the first file's name
	// at a line counted from the first file's last hunk — a file the cursor
	// was never on, at a line invented for a different one.
	t.Run("a second file's own furniture is answered with the second file", func(t *testing.T) {
		two := strings.Split(strings.TrimSuffix(twoFileDiff, "\n"), "\n")
		for _, c := range []struct {
			name string
			at   int
		}{
			{"diff --git a/webhook.go b/webhook.go", 14},
			{"index 9c1a2f0..1d4e6b3 100644", 15},
			{"--- a/webhook.go", 16},
			{"+++ b/webhook.go", 17},
		} {
			t.Run(c.name, func(t *testing.T) {
				file, line, ok := fileAt(two, c.at)
				if file != "webhook.go" || line != 1 || !ok {
					t.Errorf("fileAt(%d) = %q, %d, %v; want %q, 1, true", c.at, file, line, ok, "webhook.go")
				}
			})
		}
	})
	// The first file is still itself: a cursor inside its hunk, or on its
	// own furniture, must go on answering retry.go and not be pulled toward
	// the second file that now follows it in the same diff.
	t.Run("the first file is unaffected by a second file after it", func(t *testing.T) {
		two := strings.Split(strings.TrimSuffix(twoFileDiff, "\n"), "\n")
		if file, line, ok := fileAt(two, 9); file != "retry.go" || line != 32 || !ok {
			t.Errorf("fileAt(9) = %q, %d, %v; want %q, 32, true", file, line, ok, "retry.go")
		}
	})
	// N1: the line that broke when the boundary was decided from the prefix
	// alone. Inside a hunk, `--- ` is a removed line and nothing else, and
	// the count under it goes on being the count for the file the hunk
	// belongs to. The blast radius is what the last three rows are for: the
	// walk met the false boundary before it met the hunk header, so every
	// line below it inherited the wrong file.
	t.Run("a removed comment line is not a file boundary", func(t *testing.T) {
		sql := linesOf(commentDiff)
		for _, c := range []struct {
			name string
			at   int
			line int
		}{
			{"the removed comment itself", 6, 2},
			{"the added line under it", 7, 2},
			{"the context line under that", 8, 3},
			{"the last line of the hunk", 9, 4},
		} {
			t.Run(c.name, func(t *testing.T) {
				file, line, ok := fileAt(sql, c.at)
				if file != "schema.sql" || line != c.line || !ok {
					t.Errorf("fileAt(%d) = %q, %d, %v; want %q, %d, true", c.at, file, line, ok, "schema.sql", c.line)
				}
			})
		}
	})
	// The second file in that same diff is still reached: a fix that read
	// `--- ` as content everywhere would break the boundary I1 was raised
	// for, and this is the row that would fail if it did.
	t.Run("the file after a hunk full of comment lines is still its own", func(t *testing.T) {
		sql := linesOf(commentDiff)
		for _, at := range []int{10, 11, 12, 13} {
			if file, line, ok := fileAt(sql, at); file != "webhook.go" || line != 1 || !ok {
				t.Errorf("fileAt(%d) = %q, %d, %v; want %q, 1, true", at, file, line, ok, "webhook.go")
			}
		}
	})
	// A file git is deleting has nothing to open. Answering with the file
	// that follows it — which is what furniture without a header of its own
	// otherwise falls through to — would be the answer this pair must never
	// give: a different file, at a line the reader never asked about. o says
	// there is no file on this line instead.
	t.Run("a file being deleted is refused rather than answered with the next one", func(t *testing.T) {
		gone := linesOf(deletedFileDiff)
		for at := range 9 {
			if file, line, ok := fileAt(gone, at); ok {
				t.Errorf("fileAt(%d) = %q, %d, true; want no file at all", at, file, line)
			}
		}
		// And the file after it is unharmed.
		if file, line, ok := fileAt(gone, 12); file != "webhook.go" || line != 1 || !ok {
			t.Errorf("fileAt(12) = %q, %d, %v; want %q, 1, true", file, line, ok, "webhook.go")
		}
	})
}
