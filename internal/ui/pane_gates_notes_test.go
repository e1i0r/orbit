package ui

// pane_gates_notes_coverage_test.go covers the two panes with the least
// coverage in the whole package: gatesLines, whose only exercised path was
// the empty one, and notesLines, whose fold over EntryNoted and EntryWaiting
// had never seen either kind.

import (
	"errors"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

func TestGatesLinesAllStates(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	// 1. A record that could not be read.
	m.logErr = errors.New("record damaged")
	lines := m.gatesLines()
	if len(lines) != 1 || !strings.Contains(lines[0], "record damaged") {
		t.Errorf("gatesLines with logErr = %v, want one line naming the error", lines)
	}
	m.logErr = nil

	// 2. No gate entries at all: the empty sentence.
	m.entries = []view.Entry{{Kind: "task.created"}}
	if joined := strings.Join(m.gatesLines(), "\n"); !strings.Contains(joined, "no verification gates") {
		t.Errorf("gatesLines with no checks = %q, want the empty sentence", joined)
	}

	// 3. A mix of a pass and a fail, the fallback name and command, and the
	// reason line a failure carries.
	m.entries = []view.Entry{
		{Kind: "gate.passed", Gate: "lint", Text: "make lint"},
		{Kind: "gate.failed", Tool: "go test", Cause: "tests failed"},
	}
	joined := strings.Join(m.gatesLines(), "\n")
	for _, want := range []string{"lint", "check", "go test", "tests failed", "1/2"} {
		if !strings.Contains(joined, want) {
			t.Errorf("gatesLines mixed = %q, want it to mention %q", joined, want)
		}
	}

	// 4. Every check passed: the summary is the OK word, not the failure count.
	m.entries = []view.Entry{{Kind: "gate.passed", Gate: "vet", Text: "go vet"}}
	joined = strings.Join(m.gatesLines(), "\n")
	if !strings.Contains(joined, "1/1") {
		t.Errorf("gatesLines all-pass = %q, want the passed count to say 1/1", joined)
	}
}

func TestNotesLinesAllStates(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	// 1. A record that could not be read.
	m.logErr = errors.New("record damaged")
	lines := m.notesLines()
	if len(lines) != 1 || !strings.Contains(lines[0], "record damaged") {
		t.Errorf("notesLines with logErr = %v, want one line naming the error", lines)
	}
	m.logErr = nil

	// 2. No notes and no dialogue: the empty sentence and its hint.
	m.entries = nil
	if joined := strings.Join(m.notesLines(), "\n"); !strings.Contains(joined, "no notes or dialogue") {
		t.Errorf("notesLines empty = %q, want the empty sentence", joined)
	}

	// 3. A note with an attempt and three differently-prefixed content
	// lines, a waiting entry with only a cause, one with only text, and one
	// with neither — which the fold must drop rather than draw as a blank
	// entry.
	m.entries = []view.Entry{
		{Kind: "task.noted", Attempt: 2, Text: "→ did the thing\n[cli] ran ls\nplain note\n  \n"},
		{Kind: "phase.waiting", Phase: "gates", Cause: "blocked on review"},
		{Kind: "phase.waiting", Phase: "gates", Text: "needs a decision"},
		{Kind: "phase.waiting", Phase: "gates"},
	}
	joined := strings.Join(m.notesLines(), "\n")
	for _, want := range []string{
		"read by run 2", "did the thing", "ran ls", "plain note",
		"blocked on review", "needs a decision",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("notesLines = %q, want it to mention %q", joined, want)
		}
	}

	// 4. A note with no attempt: the plain "read by run" status, without a
	// number.
	m.entries = []view.Entry{{Kind: "task.noted", Text: "one line"}}
	joined = strings.Join(m.notesLines(), "\n")
	if !strings.Contains(joined, "read by run") || strings.Contains(joined, "read by run 0") {
		t.Errorf("notesLines with attempt 0 = %q, want the plain status", joined)
	}
}

// The other half of the dialogue: what a model or a session did to this
// task, drawn beside the notes and marked as the one thing here the next
// run is not handed.
func TestNotesLinesDrawsWhatActedFromOutsideTheRun(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.entries = []view.Entry{
		{Kind: "task.dialogue", By: "mcp", Text: "a model cancelled this task over mcp"},
		{Kind: "task.dialogue", Text: "the cockpit handed the terminal to a session"},
	}
	joined := strings.Join(m.notesLines(), "\n")
	for _, want := range []string{
		"MCP", "a model cancelled this task over mcp",
		"handed the terminal", "2 entradas",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("notesLines = %q, want it to mention %q", joined, want)
		}
	}
	if strings.Contains(joined, "read by run") {
		t.Errorf("notesLines = %q, want no claim that a run reads these", joined)
	}
}
