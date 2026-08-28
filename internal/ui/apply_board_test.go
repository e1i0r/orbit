package ui

// apply_coverage_test.go covers apply.go: the sentence helper that refuses
// an empty message, the language switch and its failure path, and the
// board-applying branches — a failed read, the first board, a crossing that
// rings the bell, and a read-failure count that only speaks when it moves.

import (
	"errors"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
)

func TestSayIgnoresAnEmptySentence(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.message, m.messageAt = "kept", m.now
	before := m.messageAt

	got := m.say("")
	if got.message != "kept" || got.messageAt != before {
		t.Errorf("say(\"\") changed the band to %+v, want it left alone", got)
	}

	got = m.say("a new sentence")
	if got.message != "a new sentence" {
		t.Errorf("say(...) = %q, want the new sentence", got.message)
	}
}

// langBadge reads a catalogue key that es.json overrides and en.json does
// not, so a test can tell which language a Printer actually resolved to
// without a Lang() accessor on the (deliberately opaque) Printer type.
func langBadge(m Model) string {
	return m.opts.Words.T("header.lang_badge", "EN")
}

func TestLanguageSwitchesAndReportsFailure(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	got := m.language("es")
	if langBadge(got) != "ES" {
		t.Errorf("language(es) left the badge %q, want ES", langBadge(got))
	}

	// A settings port that refuses the write reports the error rather than
	// rebuilding the key map.
	m.opts.Settings = &settings{fail: errors.New("disk full")}
	got = m.language("es")
	wantBand(t, got, "disk full")

	if langBadge(got) == "ES" {
		t.Error("language(es) rebuilt the key map even though the settings write failed")
	}

	// No settings port at all: the language still changes.
	m.opts.Settings = nil

	got = m.language("es")
	if langBadge(got) != "ES" {
		t.Errorf("language(es) with no settings port left the badge %q, want ES", langBadge(got))
	}
}

func TestApplyBoardFailedRead(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	before := m.board

	// A zero ReadAt with no errors is a read that simply did not happen:
	// the board on screen is kept, and nothing is said.
	next, cmd := m.applyBoard(boardMsg{})

	got := asModel(t, next)
	if got.board.ReadAt != before.ReadAt || cmd != nil {
		t.Errorf("applyBoard({}) changed the board or returned a command")
	}

	// A zero ReadAt with an error says it, and still keeps the board.
	next, _ = m.applyBoard(boardMsg{Board: board.Board{Errs: []error{errors.New("stat: permission denied")}}})
	got = asModel(t, next)
	wantBand(t, got, "permission denied")

	if got.board.ReadAt != before.ReadAt {
		t.Error("applyBoard with a read failure replaced the board on screen")
	}
}

func TestApplyBoardEnteredRingsTheBellOnlyAfterTheFirst(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// The very first board rings no bell, however it is shaped — that is
	// the guard `first` exists for — but it does place the cursor on the
	// first task.
	m.seen = false
	m.cursor = 999
	next, _ := m.applyBoard(boardMsg{
		Board:   fixtureBoard(fixtureTasks(), 4),
		Changed: board.Changed{Entered: []string{"ACME-2662"}},
	})

	got := asModel(t, next)
	if !got.seen {
		t.Error("applyBoard did not mark the board as seen")
	}

	if got.notified {
		t.Error("the first board rang the bell")
	}

	// A later board with a crossing rings the bell and says how many.
	next, cmd := got.applyBoard(boardMsg{
		Board:   fixtureBoard(fixtureTasks(), 4),
		Changed: board.Changed{Entered: []string{"ACME-2662", "ACME-2701"}},
	})

	got = asModel(t, next)
	if !got.notified {
		t.Error("a later crossing did not ring the bell")
	}

	wantBand(t, got, "need you")

	if cmd == nil {
		t.Error("a crossing produced no command at all, want at least the bell")
	}
}

func TestApplyBoardReadFailureCountOnlySpeaksWhenItMoves(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.seen = true

	board1 := board.Board{
		Tasks:  fixtureTasks(),
		ReadAt: fixtureNow,
		Errs:   []error{errors.New("first failure")},
	}
	next, _ := m.applyBoard(boardMsg{Board: board1})
	got := asModel(t, next)
	wantBand(t, got, "could not be read")

	if got.errs != 1 {
		t.Errorf("applyBoard left errs=%d, want 1", got.errs)
	}

	// The same count again: nothing new is said, so the band keeps its old
	// sentence rather than repeating the same one.
	got.message = "untouched"
	next, _ = got.applyBoard(boardMsg{Board: board1})

	got2 := asModel(t, next)
	if got2.message != "untouched" {
		t.Errorf("applyBoard repeated the read-failure count, band now %q", got2.message)
	}

	// The count dropping back to zero is worth saying too, since it is a
	// change either way.
	board0 := board.Board{Tasks: fixtureTasks(), ReadAt: fixtureNow}
	next, _ = got2.applyBoard(boardMsg{Board: board0})

	got3 := asModel(t, next)
	if got3.errs != 0 {
		t.Errorf("applyBoard left errs=%d after a clean read, want 0", got3.errs)
	}
}

func TestApplyBoardAutopilotStartsAWaitingTask(t *testing.T) {
	m, rec := testModel(t, 100, 30)
	m.seen = true
	// autopilot is on in the fixture settings; a board with a To Do task
	// and nobody waiting to be read should start it.
	_, cmd := m.applyBoard(boardMsg{Board: fixtureBoard(fixtureTasks(), 4)})
	if cmd == nil {
		t.Fatal("applyBoard under autopilot with a To Do task returned no command")
	}

	cmd()

	if rec.flow == "" {
		t.Error("autopilot did not start a task through the port")
	}
}
