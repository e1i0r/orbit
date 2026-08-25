package ui

// gesture_more_coverage_test.go is the branches gesture_test.go's own
// transition table has no reason to walk: a binding gesture does not even
// list as an affordance, a session message answering with an error or with
// nothing to run, a read that failed rather than landed, and the map
// stillTaken and took keep, exercised past the one shape each already gets
// from every other test that happens to take the keyboard or mark a task
// read.

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

func TestGestureRefusesWithNoSelectionOrNoSuchAffordance(t *testing.T) {
	// 1. Nothing under the cursor at all.
	m, _ := testModel(t, 100, 30)
	m.cursor = -1
	_, _, ok := m.gesture(m.keys.Pause)
	if ok {
		t.Error("gesture with nothing selected answered true")
	}

	// 2. The cursor is on a band header, not a task.
	m2, _ := testModel(t, 100, 30)
	m2 = at(t, m2, view.Done, true)
	_, _, ok = m2.gesture(m2.keys.Pause)
	if ok {
		t.Error("gesture on a band header answered true")
	}

	// 3. A binding that is not one of the affordances at all.
	m3, _ := testModel(t, 100, 30)
	m3 = onRow(t, m3, "ACME-2705")
	_, _, ok = m3.gesture(m3.keys.Filter)
	if ok {
		t.Error("gesture for a binding with no affordance answered true")
	}
}

func TestMarkReadKeyIsRefusedOnAnUnfinishedTask(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2705") // Running, not Done
	next, cmd := m.markReadKey()
	if cmd != nil {
		t.Error("markReadKey on a task that has not finished produced a command")
	}
	wantBand(t, asModel(t, next), "not finished")
}

func TestSessionAnswersAnErrorOrNothingToCarryOn(t *testing.T) {
	// 1. The port itself failed.
	m, _ := testModel(t, 100, 30)
	next, cmd := m.session(sessionMsg{ID: "ACME-2705", Err: errors.New("no worktree for ACME-2705")})
	if cmd != nil {
		t.Error("session with a port error produced a command")
	}
	wantBand(t, asModel(t, next), "no worktree for ACME-2705")

	// 2. The port answered with nothing to run at all.
	m2, _ := testModel(t, 100, 30)
	next2, cmd2 := m2.session(sessionMsg{ID: "ACME-2705"})
	if cmd2 != nil {
		t.Error("session with no command line produced a command")
	}
	wantBand(t, asModel(t, next2), "has no session to carry on")

	// 3. A real command line: the window suspends for it.
	m3, _ := testModel(t, 100, 30)
	next3, cmd3 := m3.session(sessionMsg{ID: "ACME-2705", Cmd: &exec.Cmd{Path: "claude"}})
	if cmd3 == nil {
		t.Fatal("session with a command line answered with no command to suspend for")
	}
	if !asModel(t, next3).taken["ACME-2705"] {
		t.Error("session did not remember that the keyboard was taken")
	}
}

func TestReadSaidReportsTheFailure(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	if got := m.readSaid(readMsg{ID: "ACME-2690", Err: errors.New("the record is damaged")}); got != "the record is damaged" {
		t.Errorf("readSaid on a failed mark = %q, want the error verbatim", got)
	}
}

func TestStillTakenDropsWhatTheBoardNoLongerHolds(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Nothing taken at all: the map is untouched.
	if after := m.stillTaken(); len(after.taken) != 0 {
		t.Errorf("stillTaken with nothing taken = %v, want empty", after.taken)
	}

	// 2. A task still parked keeps its entry; a task that finished, and one
	// that left the board altogether, both lose theirs.
	tasks := fixtureTasks()
	for i := range tasks {
		if tasks[i].ID == "ACME-2705" {
			tasks[i].Reason = view.Reason{Key: view.ReasonHeld}
		}
	}
	m2 := modelWith(t, printerFor(t, "en"), fixtureBoard(tasks, 4), 100, 30, nil)
	m2 = m2.took("ACME-2705", true) // still parked: kept
	m2 = m2.took("ACME-2706", true) // running, not parked: dropped
	m2 = m2.took("ACME-GONE", true) // not on the board at all: dropped
	after2 := m2.stillTaken()
	if !after2.taken["ACME-2705"] {
		t.Error("stillTaken forgot a task that is still parked")
	}
	if after2.taken["ACME-2706"] {
		t.Error("stillTaken kept a task that is not parked")
	}
	if after2.taken["ACME-GONE"] {
		t.Error("stillTaken kept a task no longer on the board")
	}
}

func TestTookClonesANilMapTheFirstTime(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.taken = nil // New() starts with an empty map rather than nil; force the case took must also handle
	after := m.took("ACME-2705", true)
	if !after.taken["ACME-2705"] {
		t.Error("took(id, true) on a nil map did not record the id")
	}
	released := after.took("ACME-2705", false)
	if released.taken["ACME-2705"] {
		t.Error("took(id, false) did not release the id")
	}
}
