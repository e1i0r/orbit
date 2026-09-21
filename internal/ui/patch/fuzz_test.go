package patch

// A unified diff, read against whatever git printed.
//
// The lines come from `git diff` over a worktree a model has been writing
// in, so a path may hold a space, a file may be added and deleted in the
// same diff, and a line of somebody's code may itself begin with `diff
// --git`. What this answers is drawn as a list a reader clicks, and every
// number in it is an index back into the lines it was read from — so one
// that points nowhere is a click that opens the wrong file or none.

import (
	"strings"
	"testing"
)

// FuzzAParsedDiffPointsBackAtItsOwnLines.
func FuzzAParsedDiffPointsBackAtItsOwnLines(f *testing.F) {
	for _, seed := range []string{
		"",
		"diff --git a/a.go b/a.go\n@@ -1 +1 @@\n-old\n+new",
		"diff --git a/a.go b/a.go\nnew file mode 100644\n+++ b/a.go\n@@ -0,0 +1 @@\n+first",
		"diff --git a/a.go b/a.go\ndeleted file mode 100644\n+++ /dev/null",
		"diff --git a/a.go b/a.go\ndiff --git a/b.go b/b.go",
		"diff --git",
		"diff --git a b",
		"+++ b/orphan.go\n+a line with no file",
		"diff --git a/a b/path with spaces.go",
		"@@ -1 +1 @@\n+no file at all",
	} {
		f.Add(seed)
	}

	known := map[string]bool{"NEW": true, "MOD": true, "DEL": true}

	f.Fuzz(func(t *testing.T, body string) {
		lines := strings.Split(body, "\n")

		for i, one := range Files(lines) {
			if !known[one.Status] {
				t.Errorf("file %d of %q is %q, which no reader draws", i, body, one.Status)
			}

			if one.Added < 0 || one.Deleted < 0 {
				t.Errorf("file %d of %q counts %d added and %d deleted", i, body, one.Added, one.Deleted)
			}

			// The two lines bracket the file in the diff it was read from,
			// and a pane scrolls to them: one past the end is a jump into
			// whatever happens to be there.
			if one.StartLine < 0 || one.StartLine >= len(lines) {
				t.Errorf("file %d of %q starts at line %d of %d", i, body, one.StartLine, len(lines))
			}

			if one.EndLine < one.StartLine || one.EndLine >= len(lines) {
				t.Errorf("file %d of %q runs %d..%d of %d lines",
					i, body, one.StartLine, one.EndLine, len(lines))
			}

			// Every hunk is inside the file that holds it, or the reader
			// is taken to a hunk of the file above.
			for _, at := range one.Hunks {
				if at < one.StartLine || at > one.EndLine {
					t.Errorf("file %d of %q holds a hunk at %d, outside %d..%d",
						i, body, at, one.StartLine, one.EndLine)
				}
			}
		}
	})
}

// FuzzTheTotalsAreTheSumOfTheFiles, because the header says how big the
// change is and a reader decides whether to read it on that number alone.
func FuzzTheTotalsAreTheSumOfTheFiles(f *testing.F) {
	f.Add("diff --git a/a.go b/a.go\n+one\n+two\n-three")
	f.Add("")
	f.Add("diff --git a/a.go b/a.go\n+a\ndiff --git a/b.go b/b.go\n-b")

	f.Fuzz(func(t *testing.T, body string) {
		files := Files(strings.Split(body, "\n"))

		var added, deleted int
		for _, one := range files {
			added += one.Added
			deleted += one.Deleted
		}

		gotAdded, gotDeleted := Stats(files)
		if gotAdded != added || gotDeleted != deleted {
			t.Errorf("%d files add up to +%d -%d and Stats says +%d -%d",
				len(files), added, deleted, gotAdded, gotDeleted)
		}
	})
}

// FuzzAFileAlwaysHasSomethingToCallIt, because the list draws the path and
// a row with nothing on it is a file a reader cannot pick out of the three
// beside it.
func FuzzAFileAlwaysHasSomethingToCallIt(f *testing.F) {
	f.Add("diff --git a/a.go b/a.go")
	f.Add("diff --git")
	f.Add("diff --git a/x b/")

	f.Fuzz(func(t *testing.T, body string) {
		for i, one := range Files(strings.Split(body, "\n")) {
			// Icon and Badge are what the row is drawn with, and both are
			// asked about whatever the path turned out to be.
			if Icon(one.Path) == "" {
				t.Errorf("file %d of %q (%q) is drawn with no icon", i, body, one.Path)
			}

			if Badge(one.Status) == "" {
				t.Errorf("file %d of %q is drawn with no badge for %q", i, body, one.Status)
			}
		}
	})
}
