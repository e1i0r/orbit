package ui

// Opening a file of the artifacts tab. A listing of names and sizes says
// which files a run left; it does not say what any of them holds, and what
// is inside them is the only thing this tab gives that no other tab does —
// the diff is the worktree, the timeline is the record folded into events,
// and neither is the file as it is on disk.

import (
	"errors"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// artifactFiles is the listing every test here opens against: two of the
// four files a run leaves, in the order the port answers them in.
func artifactFiles() []view.File {
	return []view.File{
		{Name: "control", Size: 5},
		{Name: "task.md", Size: 25},
	}
}

// onArtifacts is the task view on the artifacts tab, over a listing that has
// landed and a reader that will answer for the files in it.
func onArtifacts(t *testing.T, texts map[string]view.FileText, err error) (Model, *fakeReader) {
	t.Helper()

	m, r := openDetail(t, "ACME-2662")
	r.files, r.texts, r.textErr = artifactFiles(), texts, err
	m = next(t, m, filesMsg{ID: "ACME-2662", Files: r.files})

	return showing(t, m, tabArtifacts), r
}

// openFile points at one file's row and clicks it, answering the window and
// whatever the click asked for.
func openFile(t *testing.T, m Model, name string) (Model, tea.Cmd) {
	t.Helper()

	y := rowOf(screenRows(m), name)
	if y < 0 {
		t.Fatalf("%s is not on the artifacts tab:\n%s", name, strings.Join(screenRows(m), "\n"))
	}

	at := m.hit(30, y)
	if at.Kind != TargetPaneRow {
		t.Fatalf("pointing at %s = %+v, want a row of the pane", name, at)
	}

	after, cmd := m.leftClick(at)

	return asModel(t, after), cmd
}

// TestOpeningAFileShowsWhatIsInsideIt. A row that names a file and measures
// it answers "which files" and leaves "what is in them" to a reader who has
// no way to ask it from here.
func TestOpeningAFileShowsWhatIsInsideIt(t *testing.T) {
	m, r := onArtifacts(t, map[string]view.FileText{
		"task.md": {Text: "the prompt as the operator typed it\n", Whole: true},
	}, nil)

	if rowOf(screenRows(m), "the prompt as the operator typed it") >= 0 {
		t.Fatalf("the file is open before it was clicked:\n%s", strings.Join(screenRows(m), "\n"))
	}

	m, cmd := openFile(t, m, "task.md")
	if cmd == nil {
		t.Fatal("clicking a closed file asked for nothing, so the pane has nothing to show under it")
	}

	// The read is out and has not landed. An arrow that opened onto nothing
	// would read as a fold that did not work, and the reader clicks it again.
	if rowOf(screenRows(m), "opening the file") < 0 {
		t.Errorf("the file says nothing while it is being read:\n%s", strings.Join(screenRows(m), "\n"))
	}

	m = next(t, m, cmd())

	if rowOf(screenRows(m), "the prompt as the operator typed it") < 0 {
		t.Errorf("opening the file did not show what is in it:\n%s", strings.Join(screenRows(m), "\n"))
	}

	// Read to its end, so there is nothing to say about a rest of it.
	if rowOf(screenRows(m), "was not read") >= 0 {
		t.Errorf("a file read whole says part of it was not:\n%s", strings.Join(screenRows(m), "\n"))
	}

	if r.opens != 1 {
		t.Errorf("one click read the file %d times, want once", r.opens)
	}

	// Closed and opened again is the same file: it was read, and a second
	// read on every open is a read of a growing record per gesture.
	m, _ = openFile(t, m, "task.md")
	if rowOf(screenRows(m), "the prompt as the operator typed it") >= 0 {
		t.Errorf("clicking the open file did not close it:\n%s", strings.Join(screenRows(m), "\n"))
	}

	m, cmd = openFile(t, m, "task.md")
	if cmd != nil {
		t.Error("re-opening a file that was already read asked for it again")
	}

	if rowOf(screenRows(m), "the prompt as the operator typed it") < 0 {
		t.Errorf("re-opening the file did not show what the first read brought back:\n%s",
			strings.Join(screenRows(m), "\n"))
	}
}

// TestOpeningOneFileLeavesTheRestClosed. The fold is per file, so the tab
// stays a listing: a gesture that opened all of them would put the file the
// reader wanted somewhere below a record with a line per event.
func TestOpeningOneFileLeavesTheRestClosed(t *testing.T) {
	m, _ := onArtifacts(t, map[string]view.FileText{
		"task.md": {Text: "the prompt as the operator typed it\n", Whole: true},
		"control": {Text: "halted", Whole: true},
	}, nil)

	m, cmd := openFile(t, m, "task.md")
	m = next(t, m, cmd())

	if rowOf(screenRows(m), "halted") >= 0 {
		t.Errorf("opening one file opened the other:\n%s", strings.Join(screenRows(m), "\n"))
	}
}

// TestAFileReadToTheCapSaysTheRestIsNotThere. What is on screen is the start
// of a file, and a reader who takes it for the whole one reads the absence
// of a line as the line not being there.
func TestAFileReadToTheCapSaysTheRestIsNotThere(t *testing.T) {
	m, _ := onArtifacts(t, map[string]view.FileText{
		"task.md": {Text: "the first line\nthe second line\n"},
	}, nil)

	m, cmd := openFile(t, m, "task.md")
	m = next(t, m, cmd())

	rows := screenRows(m)
	if rowOf(rows, "the second line") < 0 {
		t.Fatalf("the file that was cut shows nothing at all:\n%s", strings.Join(rows, "\n"))
	}

	if rowOf(rows, "was not read") < 0 {
		t.Errorf("a file read only to the cap does not say so:\n%s", strings.Join(rows, "\n"))
	}
}

// TestAFileNobodyCanReadSaysWhy, under the row it was asked for and not in
// the band: the reader is looking at that row, and a failure said anywhere
// else is a row that stays blank for no stated reason.
func TestAFileNobodyCanReadSaysWhy(t *testing.T) {
	m, _ := onArtifacts(t, nil, errors.New("open task.md: permission denied"))

	m, cmd := openFile(t, m, "task.md")
	m = next(t, m, cmd())

	if rowOf(screenRows(m), "permission denied") < 0 {
		t.Errorf("the file that could not be read says nothing about it:\n%s", strings.Join(screenRows(m), "\n"))
	}
}

// TestAnEmptyFileSaysItIsEmpty. A file opened onto nothing reads as a fold
// that did not work, and the reader clicks it again.
func TestAnEmptyFileSaysItIsEmpty(t *testing.T) {
	m, _ := onArtifacts(t, map[string]view.FileText{"control": {Whole: true}}, nil)

	m, cmd := openFile(t, m, "control")
	m = next(t, m, cmd())

	if rowOf(screenRows(m), "is empty") < 0 {
		t.Errorf("the empty file says nothing about being empty:\n%s", strings.Join(screenRows(m), "\n"))
	}
}

// TestAFileIsShownAsItIsOnDisk. The record is one event per line and the
// marker is one field per line: a line folded onto the next would read as
// two of them, so a line too long for the pane is cut instead.
func TestAFileIsShownAsItIsOnDisk(t *testing.T) {
	long := strings.Repeat("alpha ", 200)
	m, _ := onArtifacts(t, map[string]view.FileText{"task.md": {Text: long, Whole: true}}, nil)

	m, cmd := openFile(t, m, "task.md")
	m = next(t, m, cmd())

	// One row of it, not the four the pane's width would wrap it onto.
	rows := 0

	for _, l := range m.artifactsLines() {
		if strings.Contains(l, "alpha") {
			rows++
		}
	}

	if rows != 1 {
		t.Errorf("a line of %d cells was drawn over %d rows of a pane %d wide, want one",
			len(long), rows, m.frame.Body.W)
	}

	// Cut to the pane rather than left to run off it: the pane does not wrap,
	// so a row wider than the body is a row the terminal decides the end of.
	for i, l := range m.artifactsLines() {
		if w := lipgloss.Width(ansi.Strip(l)); w > m.frame.Body.W {
			t.Errorf("row %d of the artifacts runs to %d cells on a pane of %d", i, w, m.frame.Body.W)
		}
	}
}

// TestAFileIsReadInTheSyntaxItsNameNames. What is in these files is a
// document and a word, not somebody's Go, and a well that called the record
// a language would paint half of every line as a keyword of it.
func TestAFileIsReadInTheSyntaxItsNameNames(t *testing.T) {
	for name, want := range map[string]string{
		"events.jsonl": "data",
		"run":          "",
		"control":      "",
		"task.md":      "",
	} {
		if got := fileFamily(name); got != want {
			t.Errorf("fileFamily(%q) = %q, want %q", name, got, want)
		}
	}
}

// TestAClickThatLandsAfterTheListingShrankAsksForNothing. The row a target
// carries was counted on the frame the reader clicked, and a listing that
// arrived in between is a listing that row is not in.
func TestAClickThatLandsAfterTheListingShrankAsksForNothing(t *testing.T) {
	m, _ := onArtifacts(t, map[string]view.FileText{
		"task.md": {Text: "the prompt as the operator typed it\n", Whole: true},
	}, nil)

	y := rowOf(screenRows(m), "task.md")
	if y < 0 {
		t.Fatalf("task.md is not on the artifacts tab:\n%s", strings.Join(screenRows(m), "\n"))
	}

	at := m.hit(30, y)

	// The run removed a file between the frame and the click.
	m = next(t, m, filesMsg{ID: "ACME-2662", Files: nil})

	after, cmd := m.leftClick(at)
	if cmd != nil {
		t.Error("a click on a row the listing no longer has asked for a file")
	}

	asModel(t, after)
}

// TestClosingAFileWhoseReadIsStillOutAsksForNothingMore. The gesture that
// closes a row is the same gesture that opened it, and between the two the
// answer to the first one has not come back yet.
func TestClosingAFileWhoseReadIsStillOutAsksForNothingMore(t *testing.T) {
	m, _ := onArtifacts(t, map[string]view.FileText{
		"task.md": {Text: "the prompt as the operator typed it\n", Whole: true},
	}, nil)

	m, cmd := openFile(t, m, "task.md")
	if cmd == nil {
		t.Fatal("clicking a closed file asked for nothing")
	}

	// The reply is deliberately not delivered: the read is still out.
	if _, again := openFile(t, m, "task.md"); again != nil {
		t.Error("closing a file whose read had not come back asked for it a second time")
	}
}

// TestAnOpenedRowOfAnotherPaneReadsNoFile. Every pane's rows go through the
// same opening, and only this one has a file behind them.
func TestAnOpenedRowOfAnotherPaneReadsNoFile(t *testing.T) {
	m, r := openWith(t, "ACME-2662", []view.Entry{
		{Kind: "phase.failed", Phase: "gates", Attempt: 1, Cause: wordy},
	})
	r.files = artifactFiles()
	m = next(t, m, filesMsg{ID: "ACME-2662", Files: r.files})

	// The same row number is left open on the artifacts tab, with its read
	// still out. Row nought of the timeline is row nought of the listing to
	// everything except the tab the reader is on.
	m, cmd := openFile(t, showing(t, m, tabArtifacts), "control")
	if cmd == nil {
		t.Fatal("clicking a closed file asked for nothing")
	}

	m = showing(t, m, tabTimeline)

	y := rowOf(screenRows(m), "the command ran to the end")
	if y < 0 {
		t.Fatalf("the entry that folds is not on the timeline:\n%s", strings.Join(screenRows(m), "\n"))
	}

	if _, again := m.leftClick(m.hit(30, y)); again != nil {
		t.Error("opening a row of the timeline read a file of the task's directory")
	}

	if r.opens != 0 {
		t.Errorf("the timeline read %d files, want none", r.opens)
	}
}

// TestOneFilesTextDoesNotLandOnAWindowThatNeverAskedForIt. The map is cloned
// before it is written, because a Model is copied by every method that
// returns one and a map is not: written in place, a read would show up in
// windows this one never returned.
func TestOneFilesTextDoesNotLandOnAWindowThatNeverAskedForIt(t *testing.T) {
	m, _ := onArtifacts(t, map[string]view.FileText{
		"control": {Text: "halted", Whole: true},
		"task.md": {Text: "the prompt as the operator typed it\n", Whole: true},
	}, nil)

	// One file read first, so that there is a map for the second read to
	// leak through: a nil map is replaced rather than written, which is the
	// one case an in-place write cannot be caught in.
	m, cmd := openFile(t, m, "control")
	before := next(t, m, cmd())

	after, cmd := openFile(t, before, "task.md")
	after = next(t, after, cmd())

	if _, got := before.read["task.md"]; got {
		t.Errorf("reading a file wrote it into a window that never asked: %v", before.read)
	}

	if _, got := after.read["task.md"]; !got {
		t.Errorf("the window that asked did not keep what came back: %v", after.read)
	}
}
