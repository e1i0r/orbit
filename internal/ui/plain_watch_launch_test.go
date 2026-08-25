package ui

import (
	"errors"
	"io"
	"strings"
	"testing"

	"charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

type mockRefreshReader struct {
	err     error
	board   board.Board
	changed board.Changed
}

func (r *mockRefreshReader) Refresh() (board.Board, board.Changed, error) {
	return r.board, r.changed, r.err
}

func (r *mockRefreshReader) Rescan() error {
	return nil
}

func (r *mockRefreshReader) Log(repoPath, id string) ([]view.Entry, error) {
	return nil, nil
}

func (r *mockRefreshReader) Worktree(repoPath, id string) (string, error) {
	return "", nil
}

func TestPlainFunctionEdgeCases(t *testing.T) {
	// 1. Plain with nil Reader
	rendered, err := Plain(Options{Width: 100, Height: 30})
	if err != nil {
		t.Fatalf("Plain with nil reader failed: %v", err)
	}
	if len(rendered) == 0 {
		t.Error("expected non-empty plain render with nil reader")
	}

	// 2. Plain with reader returning refresh error
	errReader := &mockRefreshReader{err: errors.New("refresh failed")}
	_, err = Plain(Options{Reader: errReader, Width: 100, Height: 30})
	if err == nil || !strings.Contains(err.Error(), "refresh failed") {
		t.Errorf("expected refresh error from Plain, got %v", err)
	}

	// 3. Plain with successful reader
	okReader := &mockRefreshReader{board: board.Board{}}
	rendered, err = Plain(Options{Reader: okReader, Width: 100, Height: 30})
	if err != nil {
		t.Fatalf("Plain with valid reader failed: %v", err)
	}
	if len(rendered) == 0 {
		t.Error("expected non-empty plain render with valid reader")
	}
}

func TestWatchScreenAndLaunchInteractions(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	// 1. launch commands that open screens
	mUpdated, _ := m.launch(Command{Name: "new"}, nil)
	m1 := asModel(t, mUpdated)
	if m1.screen != screenCompose {
		t.Errorf("launch(new) = %v, want screenCompose", m1.screen)
	}

	mUpdated, _ = m.launch(Command{Name: "settings"}, nil)
	m2 := asModel(t, mUpdated)
	if m2.screen != screenSettings {
		t.Errorf("launch(settings) = %v, want screenSettings", m2.screen)
	}

	mUpdated, _ = m.launch(Command{Name: "flows"}, nil)
	m3 := asModel(t, mUpdated)
	if m3.screen != screenFlows {
		t.Errorf("launch(flows) = %v, want screenFlows", m3.screen)
	}

	mUpdated, _ = m.launch(Command{Name: "repos"}, nil)
	m4 := asModel(t, mUpdated)
	if m4.screen != screenRepos {
		t.Errorf("launch(repos) = %v, want screenRepos", m4.screen)
	}

	// 2. runWatched command
	cmdToRun := Command{Name: "test-cmd"}
	mWatched, _ := m.runWatched(cmdToRun, []string{"arg1"})
	mWatchModel := asModel(t, mWatched)
	if !mWatchModel.watchUp || mWatchModel.watching == nil {
		t.Error("expected watchUp to be true and watching non-nil")
	}

	// watchRows rendering
	rows := mWatchModel.watchRows(10, 100)
	if len(rows) != 10 {
		t.Errorf("expected 10 watch rows, got %d", len(rows))
	}

	// Re-running while busy
	mBusy, _ := mWatchModel.runWatched(Command{Name: "other-cmd"}, nil)
	mBusyModel := asModel(t, mBusy)
	if !strings.Contains(mBusyModel.message, "still running") {
		t.Errorf("expected busy band notification, got %q", mBusyModel.message)
	}

	// watchKey Esc closes watch
	mClosed, _ := mWatchModel.watchKey(tea.KeyPressMsg{Code: tea.KeyEsc})
	mClosedModel := asModel(t, mClosed)
	if mClosedModel.watchUp {
		t.Error("expected watchUp to be false after closeWatch")
	}
}

func TestWheelAndWatchMsgCommands(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	// 1. Wheel in screenList
	mWheelDown := m.wheel(tea.Mouse{X: 10, Y: 8, Button: tea.MouseWheelDown})
	if mWheelDown.screen != screenList {
		t.Error("expected screenList")
	}
	mWheelUp := m.wheel(tea.Mouse{X: 10, Y: 8, Button: tea.MouseWheelUp})
	if mWheelUp.screen != screenList {
		t.Error("expected screenList")
	}

	// 2. Wheel in screenDetail
	m.screen = screenDetail
	m.tab = tabLog
	_ = m.wheel(tea.Mouse{X: 10, Y: 8, Button: tea.MouseWheelDown})
	_ = m.wheel(tea.Mouse{X: 10, Y: 8, Button: tea.MouseWheelUp})

	// 3. Wheel in palette and menu
	m.palette.open = true
	_ = m.wheel(tea.Mouse{X: 10, Y: 8, Button: tea.MouseWheelDown})
	m.palette.open = false
	m.menu.open = true
	_ = m.wheel(tea.Mouse{X: 10, Y: 8, Button: tea.MouseWheelDown})

	// 4. runCommand and outputPump in watchmsg
	w := &commandWatch{name: "echo-cmd"}
	cmdFn := runCommand(nil, w, []string{"arg"})
	msg := cmdFn()
	cmdMsg, ok := msg.(commandMsg)
	if !ok || cmdMsg.Err == nil {
		t.Errorf("expected commandMsg with nil port error, got %v", msg)
	}

	customDo := func(name string, args []string, out io.Writer) error {
		_, _ = out.Write([]byte("command executed\n")) //nolint:errcheck
		return nil
	}
	w2 := &commandWatch{name: "custom-cmd"}
	cmdFn2 := runCommand(customDo, w2, []string{"ok"})
	msg2 := cmdFn2()
	cmdMsg2, ok := msg2.(commandMsg)
	if !ok || cmdMsg2.Err != nil || !strings.Contains(cmdMsg2.Text, "command executed") {
		t.Errorf("unexpected custom commandMsg: %+v", cmdMsg2)
	}

	// 5. outputPump reads the watch's buffer once its tick fires — the
	// timer starts the moment outputPump is called (tea.Tick's own
	// contract), so writing to the watch first and blocking on the Cmd
	// after is what makes the write and the read the same race a real run
	// wins by construction: nothing else is writing to w.
	w3 := &commandWatch{name: "pump-cmd"}
	pump := outputPump(w3)
	_, _ = w3.Write([]byte("still going")) //nolint:errcheck
	pumpMsg, ok := pump().(outputMsg)
	if !ok || pumpMsg.Name != "pump-cmd" || !strings.Contains(pumpMsg.Text, "still going") {
		t.Errorf("outputPump's tick = %#v, want an outputMsg carrying the watch's buffer", pumpMsg)
	}
}

func TestGrownLeavesANarrowOrAlreadyTallFrameAlone(t *testing.T) {
	// 1. A window too narrow to draw is left exactly as it is.
	m, _ := testModel(t, 100, 30)
	m.tooNarrow = true
	if grown := m.grown(); grown.height != m.height {
		t.Errorf("grown() on a too-narrow window resized it to %d, want %d unchanged", grown.height, m.height)
	}

	// 2. A frame already tall enough to hold every row needs no growing.
	tall, _ := testModel(t, 100, 500)
	if grown := tall.grown(); grown.height != tall.height {
		t.Errorf("grown() on an already-tall window resized it to %d, want %d unchanged", grown.height, tall.height)
	}

	// 3. A frame shorter than the list grows to fit it, row for row.
	short, _ := testModel(t, 100, 12)
	grown := short.grown()
	if grown.height <= short.height {
		t.Errorf("grown() on a short window left height at %d, want it grown past 12", grown.height)
	}
}

func TestPlainDefaultsTheFrameWhenNoSizeIsNamed(t *testing.T) {
	rendered, err := Plain(Options{})
	if err != nil {
		t.Fatalf("Plain with no size named failed: %v", err)
	}
	if len(rendered) == 0 {
		t.Error("expected non-empty plain render with the default frame")
	}
}
