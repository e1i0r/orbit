package ui

// The file picker over a diff, drawn at every width and every position the
// cursor can be in.
//
// It is a box with a title in its top border, a list inside it and a hint
// along the bottom. Three widths are subtracted from one to size the border,
// and the pads are what keep the right edge straight — so an off-by-one here
// is a box that looks fine at the size it was written at and frays at every
// other.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/patch"
	"github.com/e1i0r/orbit/internal/words"
)

// roomForTheBox is the narrowest window the picker fits in whole. Below it
// the thirty-column content floor wins and the box is held to the window's
// edge instead.
const roomForTheBox = 40

// changed is a diff of n files, named so their widths differ.
func changed(n int) []patch.File {
	out := make([]patch.File, 0, n)
	for i := range n {
		out = append(out, patch.File{
			Path:   strings.Repeat("internal/", i%3+1) + "file.go",
			Status: []string{"NEW", "MOD", "DEL"}[i%3],
			Added:  i, Deleted: n - i,
		})
	}

	return out
}

// TestTheFilePickerHasAStraightRightEdge.
func TestTheFilePickerHasAStraightRightEdge(t *testing.T) {
	p := words.For("en")

	for _, files := range [][]patch.File{changed(1), changed(3), changed(12)} {
		for _, open := range []bool{false, true} {
			for w := 1; w <= 140; w++ {
				for at := -1; at <= len(files); at++ {
					drawn := renderDiffFileSelect(files, at, w, p, nil, open, at)
					if drawn == "" {
						continue
					}

					lines := strings.Split(drawn, "\n")

					first := lipgloss.Width(lines[0])
					for i, line := range lines {
						if got := lipgloss.Width(line); got > w {
							t.Fatalf("%d files, open=%v, w=%d, at=%d: line %d is %d cells wide",
								len(files), open, w, at, i, got)
						}
					}

					// Every line of a box is the same width, or the
					// right-hand border walks in and out as the reader
					// moves the cursor down it.
					//
					// Claimed where there is room for the box: below that
					// the content floor makes it wider than the window on
					// purpose, it is held to the edge, and where that cut
					// lands depends on the character sitting under it.
					if w < roomForTheBox {
						continue
					}

					for i, line := range lines {
						if got := lipgloss.Width(line); got != first {
							t.Fatalf("%d files, open=%v, w=%d, at=%d: line %d is %d cells and line 0 is %d",
								len(files), open, w, at, i, got, first)
						}
					}
				}
			}
		}
	}
}

// TestTheFilePickerSurvivesACursorNobodyPutThere.
//
// The index comes from a model that has been resized, had files arrive
// under it, or had a diff replaced by a shorter one — so it is regularly a
// row that is no longer there, and the picker has to draw something rather
// than reach past the end of its own list.
func TestTheFilePickerSurvivesACursorNobodyPutThere(t *testing.T) {
	p := words.For("en")
	files := changed(4)

	for _, at := range []int{-10, -1, 0, 3, 4, 99} {
		for _, open := range []bool{false, true} {
			drawn := renderDiffFileSelect(files, at, 80, p, nil, open, at)
			if drawn == "" {
				t.Errorf("a cursor at %d, open=%v, drew nothing", at, open)
			}
		}
	}

	// And a diff of no files at all, which is what a task that changed
	// nothing has.
	for _, open := range []bool{false, true} {
		if drawn := renderDiffFileSelect(nil, 0, 80, p, nil, open, 0); strings.Contains(drawn, "\x00") {
			t.Errorf("a diff of no files drew %q", drawn)
		}
	}
}

// TestACollapsedFileIsStillOnTheList, because collapsing is about the hunks
// under a file and not about the file: a reader who collapsed one and then
// could not find it would have lost it rather than tidied it.
func TestACollapsedFileIsStillOnTheList(t *testing.T) {
	p := words.For("en")
	files := changed(3)

	shut := map[string]bool{}
	for _, f := range files {
		shut[f.Path] = true
	}

	open := renderDiffFileSelect(files, 0, 100, p, shut, true, 0)
	for _, f := range files {
		if !strings.Contains(open, f.Path) {
			t.Errorf("the collapsed file %q is not on the list:\n%s", f.Path, open)
		}
	}
}
