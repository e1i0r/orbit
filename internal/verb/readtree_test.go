package verb

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
)

// TestANewFileIsOnTheMap. `git ls-files` answers with the index, and a file
// an engine wrote a minute ago is not in it. Built from that list alone the
// map dropped every new file — and a phase that creates files is the
// ordinary case, so a task whose whole change was new ones drew "this task
// has changed nothing" beside a diff showing all of them.
func TestANewFileIsOnTheMap(t *testing.T) {
	root := Grow(
		[]string{"README.md"},
		[]repo.Change{{Path: "internal/new/thing.go", Added: 40}},
	)

	if root.Changed != 1 || root.Lines != 40 {
		t.Fatalf("the root says %d files and %d lines, want 1 and 40", root.Changed, root.Lines)
	}

	if at := find(root, "internal/new/thing.go"); at == nil {
		t.Error("the new file is not on the map")
	}
}

// TestABinaryWeighsNothing, so that one image does not outweigh a rewritten
// package. git counts no lines for it because there are none.
func TestABinaryWeighsNothing(t *testing.T) {
	root := Grow(
		[]string{"logo.png", "main.go"},
		[]repo.Change{
			{Path: "logo.png", Added: -1, Deleted: -1},
			{Path: "main.go", Added: 3, Deleted: 1},
		},
	)

	if root.Changed != 2 {
		t.Errorf("%d files counted as touched, want 2", root.Changed)
	}

	if root.Lines != 4 {
		t.Errorf("the change weighs %d, want 4 — the image should weigh nothing", root.Lines)
	}
}

// TestOnlySiblingsAreSeatedTogether. The neighbours carried to a surface are
// what a level can put side by side, and a level is the children of one
// directory.
func TestOnlySiblingsAreSeatedTogether(t *testing.T) {
	root := Grow(
		[]string{"internal/a/x.go", "internal/b/y.go", "site/z.html"},
		nil,
		repo.Near{A: "internal/a", B: "internal/b", Times: 9},
	)

	at := find(root, "internal/a")
	if at == nil || len(at.With) != 1 || at.With[0].Path != "internal/b" {
		t.Fatalf("internal/a was seated with %v", at)
	}

	if at.With[0].Times != 9 {
		t.Errorf("the pair came through as %d, want 9", at.With[0].Times)
	}
}

// find is the cell one path names, anywhere under the root.
func find(at Cell, path string) *Cell {
	if at.Path == path {
		return &at
	}

	for i := range at.Cells {
		if got := find(at.Cells[i], path); got != nil {
			return got
		}
	}

	return nil
}

// TestOneDirectoryIsOneCellHoweverManyFilesAreUnderIt.
//
// The tree is walked segment by segment and a segment already there is
// descended into rather than added again. A second cell of the same name is
// a directory drawn twice, each holding half of what is in it, and the
// weight the map draws with split between them.
func TestOneDirectoryIsOneCellHoweverManyFilesAreUnderIt(t *testing.T) {
	root := Grow([]string{"internal/db/pr.go", "internal/db/rule.go", "internal/ui/wheel.go"}, nil)

	if len(root.Cells) != 1 || root.Cells[0].Name != "internal" {
		t.Fatalf("the root holds %d cells, want the one directory they are all under", len(root.Cells))
	}

	inside := root.Cells[0]
	if len(inside.Cells) != 2 {
		t.Fatalf("internal holds %d cells, want db and ui", len(inside.Cells))
	}

	if len(inside.Cells[0].Cells) != 2 {
		t.Errorf("db holds %d files, want the two written under it", len(inside.Cells[0].Cells))
	}
}

// TestTheMapReadsTheSameWayTwice.
//
// Directories first, because they are where the reader goes next, and then
// by name. The same tree is drawn on a terminal and in a browser, and a map
// whose cells move between two readings is one nobody can learn the shape
// of.
func TestTheMapReadsTheSameWayTwice(t *testing.T) {
	root := Grow([]string{"zebra.md", "internal/b/y.go", "apple.md", "alpha/x.go"}, nil)

	want := []string{"alpha", "internal", "apple.md", "zebra.md"}
	if len(root.Cells) != len(want) {
		t.Fatalf("the root holds %d cells, want %d", len(root.Cells), len(want))
	}

	for i, name := range want {
		if root.Cells[i].Name != name {
			t.Errorf("cell %d is %q, want %q — directories first, then by name",
				i, root.Cells[i].Name, name)
		}
	}
}

// TestWhatTheTaskTouchedIsMarkedAndNothingElseIs, because the mark is the
// whole of what the drawing says: a map with everything lit says the task
// changed the repository, and one with nothing lit says it changed nothing.
func TestWhatTheTaskTouchedIsMarkedAndNothingElseIs(t *testing.T) {
	root := Grow(
		[]string{"touched.go", "untouched.go"},
		[]repo.Change{{Path: "touched.go", Added: 3}},
	)

	lines := strings.Split(drawn(root), "\n")
	if len(lines) != 2 {
		t.Fatalf("a checkout of two files drew %d lines: %q", len(lines), drawn(root))
	}

	if lines[0] != " ● touched.go" {
		t.Errorf("the file the task changed reads %q", lines[0])
	}

	if lines[1] != "   untouched.go" {
		t.Errorf("a file the task never reached reads %q", lines[1])
	}
}
