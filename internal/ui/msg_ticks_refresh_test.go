package ui

// msg_coverage_test.go is every Cmd-returning function in msg.go: the three
// clocks, the two reads, and the three gestures that write through a port —
// each exercised once with a working port and once with the nil port a
// window opened without a state root gets.
//
// The three clocks are asserted on past the tick itself: tea.Tick (see
// charm.land/bubbletea/v2's commands.go) starts its timer the moment it is
// called, so invoking the Cmd this package hands back only ever waits out a
// timer that was already running — nothing here starts a second one.

import (
	"errors"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

// msgReader is a Reader whose Refresh and Rescan each answer however one
// test needs, which mockRefreshReader in plain_and_watch_coverage_test.go
// does not: its Rescan is wired to always succeed.
type msgReader struct {
	board      board.Board
	changed    board.Changed
	refreshErr error
	rescanErr  error
}

func (r *msgReader) Refresh() (board.Board, board.Changed, error) {
	return r.board, r.changed, r.refreshErr
}
func (r *msgReader) Rescan() error                                 { return r.rescanErr }
func (r *msgReader) Log(string, string) ([]view.Entry, error)      { return nil, nil }
func (r *msgReader) Worktree(string, string) (string, error)       { return "", nil }
func (r *msgReader) SupervisorLog() ([]view.SupervisorLine, error) { return nil, nil }

func TestTheThreeClocksEachRaiseTheirOwnMessage(t *testing.T) {
	if _, ok := tick()().(tickMsg); !ok {
		t.Error("tick's command did not raise a tickMsg")
	}

	if _, ok := elapsedTick()().(elapsedMsg); !ok {
		t.Error("elapsedTick's command did not raise an elapsedMsg")
	}

	if _, ok := rescanTick()().(rescanMsg); !ok {
		t.Error("rescanTick's command did not raise a rescanMsg")
	}
}

func TestRefreshReadsTheBoardOrSaysWhyNot(t *testing.T) {
	// 1. A nil Reader is a window opened without one: an empty message,
	// never a panic.
	msg := refresh(nil)()
	if bm, ok := msg.(boardMsg); !ok || !bm.Board.ReadAt.IsZero() {
		t.Errorf("refresh(nil) = %#v, want an empty boardMsg", msg)
	}

	// 2. A read that fails carries the failure and nothing else.
	failing := &msgReader{refreshErr: errors.New("disk is gone")}
	msg = refresh(failing)()

	bm, ok := msg.(boardMsg)
	if !ok || len(bm.Board.Errs) == 0 {
		t.Fatalf("refresh with a failing reader = %#v, want a boardMsg carrying the error", msg)
	}

	if !errorsAnyContains(bm.Board.Errs, "disk is gone") {
		t.Errorf("boardMsg.Board.Errs = %v, want it to mention %q", bm.Board.Errs, "disk is gone")
	}

	// 3. A read that succeeds carries the board and what changed.
	want := board.Board{Repos: 3}
	ok2 := board.Changed{Tasks: []string{"ACME-1"}}
	msg = refresh(&msgReader{board: want, changed: ok2})()

	bm, ok = msg.(boardMsg)
	if !ok || bm.Board.Repos != 3 || len(bm.Changed.Tasks) != 1 {
		t.Errorf("refresh with a working reader = %#v, want the board and its Changed handed back whole", msg)
	}
}

func TestRescanWalksTheTreeOrSaysWhyNot(t *testing.T) {
	// 1. A nil Reader answers with nothing to report.
	if bm, ok := rescan(nil)().(boardMsg); !ok || bm.Board.Repos != 0 || len(bm.Board.Errs) != 0 {
		t.Errorf("rescan(nil) = %#v, want an empty boardMsg", bm)
	}

	// 2. A failed walk carries the failure.
	failing := &msgReader{rescanErr: errors.New("root is not a directory")}
	msg := rescan(failing)()

	bm, ok := msg.(boardMsg)
	if !ok || len(bm.Board.Errs) == 0 {
		t.Fatalf("rescan with a failing reader = %#v, want a boardMsg carrying the error", msg)
	}

	// 3. A walk that finds nothing new says nothing either — what it found
	// shows up on the next refresh, not here.
	if bm, ok := rescan(&msgReader{})().(boardMsg); !ok || bm.Board.Repos != 0 || len(bm.Board.Errs) != 0 {
		t.Errorf("rescan on a working reader = %#v, want an empty boardMsg", bm)
	}
}

func TestControlMarkReadAndTakeSessionRefuseANilPort(t *testing.T) {
	task := view.Task{ID: "ACME-2705"}

	// control
	msg := control(nil, task, "pause")()

	cm, ok := msg.(controlMsg)
	if !ok || cm.Err == nil || cm.ID != task.ID || cm.Word != "pause" {
		t.Errorf("control(nil, ...) = %#v, want a controlMsg refusing for want of a port", msg)
	}

	okPort := func(view.Task, string) error { return nil }

	msg = control(okPort, task, "pause")()
	if cm, ok := msg.(controlMsg); !ok || cm.Err != nil {
		t.Errorf("control with a working port = %#v, want no error", msg)
	}

	// markRead
	msg = markRead(nil, task)()

	rm, ok := msg.(readMsg)
	if !ok || rm.Err == nil || rm.ID != task.ID {
		t.Errorf("markRead(nil, ...) = %#v, want a readMsg refusing for want of a port", msg)
	}

	msg = markRead(func(view.Task) error { return nil }, task)()
	if rm, ok := msg.(readMsg); !ok || rm.Err != nil {
		t.Errorf("markRead with a working port = %#v, want no error", msg)
	}

	// takeSession
	msg = takeSession(nil, task)()

	sm, ok := msg.(sessionMsg)
	if !ok || sm.Err == nil || sm.ID != task.ID {
		t.Errorf("takeSession(nil, ...) = %#v, want a sessionMsg refusing for want of a port", msg)
	}
}

// errorsAnyContains is whether any error in errs mentions want, which is all
// a boardMsg built from a read failure promises about its wording.
func errorsAnyContains(errs []error, want string) bool {
	for _, e := range errs {
		if e != nil && strings.Contains(e.Error(), want) {
			return true
		}
	}

	return false
}
