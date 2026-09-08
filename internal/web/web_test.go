package web

// What the routes answer, asked through the handler rather than through the
// functions under it: the page reads HTTP, so that is what is tested.

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

// aBoard is a reader with the rows a test wants and nothing behind them.
type aBoard struct {
	tasks   []view.Task
	entries []view.Entry
	err     error
	logErr  error
}

func (b aBoard) Refresh() (board.Board, board.Changed, error) {
	if b.err != nil {
		return board.Board{}, board.Changed{}, b.err
	}

	out := board.Board{Tasks: b.tasks, ReadAt: time.Now()}
	for _, t := range b.tasks {
		out.Counts[view.BandOf(t)]++
	}

	return out, board.Changed{}, nil
}

func (b aBoard) Log(_, _ string) ([]view.Entry, error) { return b.entries, b.logErr }

// nowhere is a worktree port that answers a path nothing is at.
type nowhere struct{ path string }

func (n nowhere) Worktree(_, _ string) (string, error) {
	if n.path == "" {
		return "", errors.New("no state root")
	}

	return n.path, nil
}

func ask(t *testing.T, s *Server, path string) (int, map[string]any) {
	t.Helper()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	var body map[string]any

	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("%s answered something that is not JSON: %v\n%s", path, err, rec.Body.String())
	}

	return rec.Code, body
}

// listIn is one list off an answer, and a failed test rather than an empty
// slice when the field is not one.
func listIn(t *testing.T, body map[string]any, field string) []any {
	t.Helper()

	out, ok := body[field].([]any)
	if !ok {
		t.Fatalf("%q came back as %T: %v", field, body[field], body)
	}

	return out
}

// TestTheBoardIsAnsweredWithItsBandsAndRows.
func TestTheBoardIsAnsweredWithItsBandsAndRows(t *testing.T) {
	s := New(aBoard{tasks: []view.Task{
		{ID: "LED-1", Title: "fix the total", Band: view.ToDo, Repo: "ledger"},
		{ID: "LED-2", Title: "and the other one", Band: view.Done, Repo: "ledger"},
	}}, nowhere{path: "/nowhere"}, "/code")

	code, body := ask(t, s, "/api/board")
	if code != http.StatusOK {
		t.Fatalf("the board answered %d: %v", code, body)
	}

	if got := body["root"]; got != "/code" {
		t.Errorf("the board says its root is %v", got)
	}

	rows := listIn(t, body, "tasks")
	if len(rows) != 2 {
		t.Fatalf("the board carries %d rows, want the two it was given", len(rows))
	}

	bands := listIn(t, body, "bands")
	if len(bands) != 4 {
		t.Fatalf("the board carries %d bands, want the four there are", len(bands))
	}

	// The bands are named and not numbered, because the numbers are
	// internal/view's to renumber and a page served yesterday still reads.
	first, ok := bands[0].(map[string]any)
	if !ok {
		t.Fatalf("a band came back as %T", bands[0])
	}

	if first["name"] != "needs_you" {
		t.Errorf("the first band is %v, want needs_you", first["name"])
	}
}

// TestATaskCarriesItsRecord.
func TestATaskCarriesItsRecord(t *testing.T) {
	s := New(aBoard{
		tasks:   []view.Task{{ID: "LED-1", Title: "fix the total", Band: view.Done}},
		entries: []view.Entry{{Kind: "task.created", Text: "fix the total"}, {Kind: "task.finished"}},
	}, nowhere{path: "/nowhere"}, "/code")

	code, body := ask(t, s, "/api/tasks/LED-1")
	if code != http.StatusOK {
		t.Fatalf("the task answered %d: %v", code, body)
	}

	entries := listIn(t, body, "entries")
	if len(entries) != 2 {
		t.Fatalf("the task carries %d entries, want the two the record holds", len(entries))
	}
}

// TestATaskNobodyHasIsRefusedAndNotInvented.
func TestATaskNobodyHasIsRefusedAndNotInvented(t *testing.T) {
	s := New(aBoard{}, nowhere{path: "/nowhere"}, "/code")

	code, body := ask(t, s, "/api/tasks/NOPE-1")
	if code != http.StatusNotFound {
		t.Fatalf("a task nobody has answered %d: %v", code, body)
	}

	if body["error"] == nil {
		t.Errorf("the refusal says nothing: %v", body)
	}
}

// TestATaskWithNoCheckoutIsMissingRatherThanFailed.
//
// The ordinary state of every row in To Do. Asked as a failure, git answers
// with a chdir error about a path the reader never chose and cannot act on.
func TestATaskWithNoCheckoutIsMissingRatherThanFailed(t *testing.T) {
	s := New(aBoard{tasks: []view.Task{{ID: "LED-1", RepoPath: "/code/ledger", Band: view.ToDo}}},
		nowhere{path: "/nowhere-at-all"}, "/code")

	code, body := ask(t, s, "/api/tasks/LED-1/diff")
	if code != http.StatusOK {
		t.Fatalf("the diff answered %d: %v", code, body)
	}

	if body["missing"] != true {
		t.Errorf("a task with no checkout answered %v", body)
	}

	if body["failed"] != nil {
		t.Errorf("a task with no checkout reported a failure: %v", body["failed"])
	}
}

// TestATaskInNoRepositoryHasNothingToDiff.
func TestATaskInNoRepositoryHasNothingToDiff(t *testing.T) {
	s := New(aBoard{tasks: []view.Task{{ID: "LED-1", Band: view.ToDo}}}, nowhere{}, "/code")

	code, body := ask(t, s, "/api/tasks/LED-1/diff")
	if code != http.StatusOK || body["missing"] != true {
		t.Errorf("a task in no repository answered %d: %v", code, body)
	}
}

// TestABoardThatWillNotReadSaysSo, rather than answering an empty board —
// which is a different picture and a true one.
func TestABoardThatWillNotReadSaysSo(t *testing.T) {
	s := New(aBoard{err: errors.New("the record is locked")}, nowhere{}, "/code")

	code, body := ask(t, s, "/api/board")
	if code != http.StatusInternalServerError {
		t.Fatalf("a board that would not read answered %d: %v", code, body)
	}

	if body["error"] == nil {
		t.Errorf("the failure says nothing: %v", body)
	}
}

// TestThePageIsServedAtTheRootAndNowhereElse.
func TestThePageIsServedAtTheRootAndNowhereElse(t *testing.T) {
	s := New(aBoard{}, nowhere{}, "/code")

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("the page answered %d", rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("the page is served as %q", got)
	}

	other := httptest.NewRecorder()
	s.Handler().ServeHTTP(other, httptest.NewRequest(http.MethodGet, "/nope", nil))

	if other.Code != http.StatusNotFound {
		t.Errorf("a path that is not the page answered %d", other.Code)
	}
}
