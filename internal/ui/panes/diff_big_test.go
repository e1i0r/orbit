package panes

// A diff too big to draw, which is an ordinary thing for a repository to
// have: generated files — API docs, protobuf, mocks, lockfiles, snapshots —
// are rewritten whole by the tools that make them.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/patch"
	"github.com/e1i0r/orbit/internal/words"
)

// reformatted is the shape reported in #133: a codegen step that rewrote
// three files end to end, 126,000 lines between them.
func reformatted(files, linesPer int) string {
	var b strings.Builder

	for f := range files {
		fmt.Fprintf(&b, "diff --git a/gen/%d.go b/gen/%d.go\n", f, f)
		fmt.Fprintf(&b, "--- a/gen/%d.go\n+++ b/gen/%d.go\n", f, f)
		fmt.Fprintf(&b, "@@ -1,%d +1,%d @@\n", linesPer, linesPer)

		for i := range linesPer {
			fmt.Fprintf(&b, "+  line %d of file %d\n-  old line %d\n", i, f, i)
		}
	}

	return b.String()
}

// TestAHugeFileKeepsItsCardAndLosesItsHunks. Every line drawn is a line
// lipgloss styles, and the window redraws twice a second: drawing all of
// them took 303ms a frame, which is a cockpit that does not answer the
// keyboard. The list, the counts and the navigation are what a reader
// actually uses on a file that size.
func TestAHugeFileKeepsItsCardAndLosesItsHunks(t *testing.T) {
	huge := reformatted(3, 21000)
	lines, files := formatStructuredDiff(huge, 100, words.For("en"), nil, false, nil, false)

	if len(files) != 3 {
		t.Fatalf("%d files parsed, want 3", len(files))
	}

	drawn := strings.Join(lines, "\n")
	for _, want := range []string{"gen/0.go", "gen/1.go", "gen/2.go", "not drawn"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the diff does not say %q", want)
		}
	}

	if len(lines) > mostLinesDrawn {
		t.Errorf("%d lines drawn, and the budget is %d", len(lines), mostLinesDrawn)
	}
}

// TestASmallDiffIsDrawnWhole, because the cap is for the diff nobody can
// read and not for the one everybody does.
func TestASmallDiffIsDrawnWhole(t *testing.T) {
	lines, _ := formatStructuredDiff(reformatted(2, 20), 100, words.For("en"), nil, false, nil, false)

	drawn := strings.Join(lines, "\n")
	if strings.Contains(drawn, "not drawn") {
		t.Errorf("a 80-line diff was cut:\n%s", drawn)
	}

	if !strings.Contains(drawn, "line 19 of file 1") {
		t.Error("the last line of the last file is not drawn")
	}
}

// TestTheBudgetIsSpentFrontToBack: what a reader sees drawn is the front of
// their diff, not whichever files happened to be small.
func TestTheBudgetIsSpentFrontToBack(t *testing.T) {
	files := []struct {
		path  string
		lines int
	}{{"a", 1500}, {"b", 1500}, {"c", 1500}}

	var b strings.Builder

	for _, f := range files {
		fmt.Fprintf(&b, "diff --git a/%s b/%s\n", f.path, f.path)
		fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n@@ -1,1 +1,1 @@\n", f.path, f.path)

		for i := range f.lines {
			fmt.Fprintf(&b, "+  %d\n", i)
		}
	}

	parsed := undrawn(patchFiles(b.String()))
	if len(parsed) != 0 {
		t.Errorf("%v left undrawn, and all three fit inside the budget", parsed)
	}
}

// patchFiles is the diff as the parser sees it.
func patchFiles(text string) []patch.File {
	return patch.Files(strings.Split(strings.TrimSuffix(text, "\n"), "\n"))
}

// BenchmarkFormatBigDiff is the measurement the cap was chosen from: 303ms a
// frame before it, single-digit milliseconds after.
func BenchmarkFormatBigDiff(b *testing.B) {
	text := reformatted(3, 21000)

	b.ResetTimer()

	for range b.N {
		formatStructuredDiff(text, 100, words.For("en"), nil, false, nil, false)
	}
}
