package ui

// reader_test.go is the port the task view reads through, satisfied in
// memory, and the record it reads.
//
// internal/ui declares Reader as an interface precisely so that a test can
// answer it without a state root anywhere in the picture: no directory is
// walked, no log is opened, and the guard in main_test.go stays true. The
// fixture repository the diff is actually taken from is in gitrepo_test.go,
// because a real git repository is a different kind of fixture from a value
// and only one test needs it.

import (
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// fakeReader answers the six questions Reader asks, from values a test wrote
// out. It counts the reads of the log and of a file so that a test can say
// whether either is being taken again rather than only whether it looks
// right.
type fakeReader struct {
	board    board.Board
	changed  board.Changed
	entries  []view.Entry
	worktree string
	logErr   error
	treeErr  error
	files    []view.File
	filesErr error
	texts    map[string]view.FileText
	textErr  error
	reads    int
	opens    int
}

func (f *fakeReader) Refresh() (board.Board, board.Changed, error) {
	return f.board, f.changed, nil
}

func (f *fakeReader) Rescan() error { return nil }

func (f *fakeReader) Files(_, _ string) ([]view.File, error) {
	return f.files, f.filesErr
}

func (f *fakeReader) FileText(_, _, name string) (view.FileText, error) {
	f.opens++

	return f.texts[name], f.textErr
}

func (f *fakeReader) Log(_, _ string) ([]view.Entry, error) {
	f.reads++
	if f.logErr != nil {
		return nil, f.logErr
	}

	return f.entries, nil
}

func (f *fakeReader) Worktree(_, _ string) (string, error) {
	if f.treeErr != nil {
		return "", f.treeErr
	}

	return f.worktree, nil
}

func (f *fakeReader) SupervisorLog() ([]view.SupervisorLine, error) {
	return nil, nil
}

// fixtureEntries is the record of ACME-2662, which fixtureTasks bands as
// needing you after two attempts. It is written out rather than folded from
// a log on disk for the same reason the board fixture is: what the pane
// draws has to be a fact about the fixture and not about the machine.
//
// The two attempts are the point. The first ran one phase through and broke
// in the next; the second broke in the same place. That is what puts a seam
// in the middle of the log tab and two blocks in the evidence tab, and it is
// the state a reader most often opens this screen in.
func fixtureEntries() []view.Entry {
	return []view.Entry{
		{
			At: ago(70 * time.Minute), Kind: "task.created", Attempt: 0,
			Text: "Retry the webhook on 5xx",
		},
		{At: ago(66 * time.Minute), Kind: "task.started", Attempt: 1},
		{
			At: ago(66 * time.Minute), Kind: "phase.started", Phase: "implement", Attempt: 1,
			PhaseN: 1, Engine: "claude", Model: "opus",
		},
		{
			At: ago(52 * time.Minute), Kind: "phase.finished", Phase: "implement", Attempt: 1,
			PhaseN: 1, Cost: 0.42, Session: "8f2c31", Kept: 30,
			Text: "wrote retry.go\nadded a backoff",
		},
		{
			At: ago(52 * time.Minute), Kind: "phase.started", Phase: "gates", Attempt: 1,
			PhaseN: 2, Engine: "claude", Model: "opus",
		},
		{
			At: ago(49 * time.Minute), Kind: "phase.failed", Phase: "gates", Attempt: 1,
			PhaseN: 2, Cost: 0.11, Session: "8f2c31", Kept: 37,
			Text: "go vet: retry.go:31: unreachable code", Cause: "the gates phase exited 1",
		},
		{At: ago(49 * time.Minute), Kind: "task.failed", Attempt: 1, Text: "gates did not pass"},
		{At: ago(34 * time.Minute), Kind: "task.started", Attempt: 2},
		{
			At: ago(34 * time.Minute), Kind: "phase.started", Phase: "gates", Attempt: 2,
			PhaseN: 2, Engine: "claude", Model: "opus",
		},
		{
			At: ago(31 * time.Minute), Kind: "phase.failed", Phase: "gates", Attempt: 2,
			PhaseN: 2, Cost: 0.09, Session: "b41d07", Kept: 37, Full: 1048583,
			Text: "go vet: retry.go:31: unreachable code", Cause: "the gates phase exited 1",
		},
		{At: ago(31 * time.Minute), Kind: "task.failed", Attempt: 2, Text: "gates did not pass"},
	}
}

// fixtureDiff is what git said about the fixture task's worktree, written
// out. The goldens are drawn from this and not from a live git for the same
// reason they are drawn from a fixed clock: an index hash in a golden file
// is a fact about the machine that regenerated it.
const fixtureDiff = `diff --git a/retry.go b/retry.go
index 3f1c9a2..a77b104 100644
--- a/retry.go
+++ b/retry.go
@@ -28,7 +28,12 @@ func send(req *http.Request) error {
 	resp, err := do(req)
 	if err != nil {
 		return err
 	}
+	if resp.StatusCode >= 500 {
+		return backoff(req, resp)
+	}
 	return nil
 }
`

// twoFileDiff is the same change to retry.go, in a diff that also touches a
// second file. It exists for one reason: every other fixture in this file
// is a single file, and fileAt's walk from the cursor up to the file it
// belongs to has a failure only a second file can expose — a line in the
// furniture that introduces file two, met before file one's own hunk is,
// answered with file one's name and a line number counted from file one's
// last hunk.
const twoFileDiff = `diff --git a/retry.go b/retry.go
index 3f1c9a2..a77b104 100644
--- a/retry.go
+++ b/retry.go
@@ -28,7 +28,12 @@ func send(req *http.Request) error {
 	resp, err := do(req)
 	if err != nil {
 		return err
 	}
+	if resp.StatusCode >= 500 {
+		return backoff(req, resp)
+	}
 	return nil
 }
diff --git a/webhook.go b/webhook.go
index 9c1a2f0..1d4e6b3 100644
--- a/webhook.go
+++ b/webhook.go
@@ -1,3 +1,3 @@
 package webhook
-func old() {}
+func retry() {}
`

// tallDiff is the same change with a hunk long enough to run past any pane
// on the sizes this suite renders at. A diff that fits on screen cannot be
// scrolled, and a test over one would say nothing about where the editor
// opens — the answer would be the top of the file whatever the rule did.
func tallDiff() string {
	var b strings.Builder
	b.WriteString("diff --git a/retry.go b/retry.go\n")
	b.WriteString("index 3f1c9a2..a77b104 100644\n")
	b.WriteString("--- a/retry.go\n")
	b.WriteString("+++ b/retry.go\n")
	b.WriteString("@@ -28,60 +28,60 @@ func send(req *http.Request) error {\n")

	for i := range 60 {
		b.WriteString(" \tattempt(" + strconv.Itoa(i) + ")\n")
	}

	return b.String()
}

// wideDiff is a change to a file that has no newlines worth speaking of,
// which is the case sideways scrolling exists for: a minified bundle, a
// generated lockfile, a long JSON literal. The pane can only scroll along a
// line that is wider than the pane, so a test over the ordinary fixture
// would pass no matter what ← and → did.
func wideDiff() string {
	return "diff --git a/bundle.js b/bundle.js\n--- a/bundle.js\n+++ b/bundle.js\n@@ -1 +1 @@\n-" +
		strings.Repeat("a", 4000) + "\n+" + strings.Repeat("b", 4000) + "\n"
}

// step delivers one keystroke and returns the window it left behind, so a
// test that walks four keys reads as four keys rather than as four type
// assertions.
func step(t *testing.T, m Model, keystroke string) Model {
	t.Helper()
	return next(t, m, press(keystroke))
}

// next delivers one message and returns the window it left behind.
func next(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()

	after, _ := m.Update(msg)

	got, ok := after.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", after)
	}

	return got
}

// onto puts the cursor on one task's row, which is the precondition every
// gesture into the task view arrives under.
func onto(t *testing.T, m Model, id string) Model {
	t.Helper()

	for i, r := range m.rows() {
		if !r.head && !r.blank && r.task.ID == id {
			m.cursor = i
			return m
		}
	}

	t.Fatalf("no row for task %s in the body", id)

	return m
}

// openDetail is the fixture window with the task view open on one id and
// both panes filled, reached through the gesture that opens it rather than
// by writing the fields — so that what a test asserts is what a reader would
// have got. The two answers are delivered as messages because that is how
// they arrive: open asks, and the reply lands one loop later.
func openDetail(t *testing.T, id string) (Model, *fakeReader) {
	t.Helper()
	return openIn(t, words.For("en"), id, fixtureEntries(), fixtureDiff)
}

// openWith is the same window over a record a test chose.
func openWith(t *testing.T, id string, entries []view.Entry) (Model, *fakeReader) {
	t.Helper()
	return openIn(t, words.For("en"), id, entries, fixtureDiff)
}

// openIn is the general form: a language, a record and a diff.
func openIn(t *testing.T, p *words.Printer, id string, entries []view.Entry, diff string) (Model, *fakeReader) {
	t.Helper()

	r := &fakeReader{entries: entries, worktree: "/w/" + id}
	m := modelWith(t, p, fixtureBoard(fixtureTasks(), 4), 100, 30, nil)
	m.opts.Reader = r
	m = step(t, onto(t, m, id), "enter")
	m = next(t, m, logMsg{ID: id, Entries: r.entries})

	return next(t, m, diffMsg{ID: id, Text: diff, Tree: r.worktree}), r
}

// showing tabs round to the pane a test is about, through the key that does
// it rather than by writing the field.
func showing(t *testing.T, m Model, which tab) Model {
	t.Helper()

	for range int(tabCount) {
		if m.tab == which {
			return m
		}

		m = step(t, m, "tab")
	}

	t.Fatalf("the task view will not tab round to %v", which)

	return m
}

// longLog is the fixture record with enough filler in it to run past the
// pane. It is the precondition every follow-and-release row arrives under: a
// log that fits on screen can be neither scrolled nor followed, so a test
// over the short one would pass whatever the rule did.
func longLog() []view.Entry {
	entries := fixtureEntries()
	for i := range 40 {
		entries = append(entries, view.Entry{
			At: ago(time.Duration(30-i) * time.Minute), Kind: "phase.started",
			Phase: "gates", Attempt: 2, PhaseN: 2, Engine: "claude", Model: "opus",
		})
	}

	return entries
}

// paneText is the whole frame the task view is drawing, stripped of colour,
// as one string. It is how a test says that a sentence landed in the pane
// rather than in the activity band.
func paneText(t *testing.T, m Model) string {
	t.Helper()
	return strings.Join(renderAt(t, m, 100, 30), "\n")
}

// wantIn fails unless text mentions want, and prints the frame when it does
// not — a pane assertion that fails without showing the pane costs a rerun.
func wantIn(t *testing.T, text, want string) {
	t.Helper()

	if !strings.Contains(text, want) {
		t.Errorf("the task view drew:\n%s\nwant it to mention %q", text, want)
	}
}
