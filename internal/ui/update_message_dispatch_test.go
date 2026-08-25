package ui

// update_message_coverage_test.go drives Update's own dispatcher through
// every case a rendering or keyboard test does not otherwise reach: the
// background colour probe, a watch's output and verdict messages under
// every combination of "is this still the watch we asked for", and the
// interactive-CLI and editor messages' own branches.

import (
	"errors"
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestUpdateBackgroundColorMsg(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	next, cmd := m.Update(tea.BackgroundColorMsg{Color: color.Black})
	got := asModel(t, next)
	if cmd != nil {
		t.Error("BackgroundColorMsg produced a command, want none")
	}
	if !got.dark {
		t.Error("a black background did not set m.dark")
	}
}

func TestUpdateOutputMsgFollowsOnlyItsOwnWatch(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. No watch at all: dropped.
	next, cmd := m.Update(outputMsg{Name: "sync", Text: "line one"})
	got := asModel(t, next)
	if cmd != nil || got.output != "" {
		t.Error("outputMsg with no watch running was not dropped")
	}

	// 2. A watch running, but for a different command: dropped.
	m.watching = &commandWatch{name: "other"}
	next, cmd = m.Update(outputMsg{Name: "sync", Text: "line one"})
	got = asModel(t, next)
	if cmd != nil || got.output != "" {
		t.Error("outputMsg for a stale watch was not dropped")
	}

	// 3. The watch this window is actually running: the text lands and the
	// pump is re-armed.
	m.watching = &commandWatch{name: "sync"}
	next, cmd = m.Update(outputMsg{Name: "sync", Text: "line one"})
	got = asModel(t, next)
	if got.output != "line one" || cmd == nil {
		t.Errorf("outputMsg for the live watch = output %q cmd-nil %v, want it taken and the pump re-armed",
			got.output, cmd == nil)
	}
}

func TestUpdateCommandMsgEveryBranch(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. The watch this window is running finishes cleanly: output is
	// taken, the watch is cleared, and the band says it is done.
	m.watching = &commandWatch{name: "sync"}
	next, _ := m.Update(commandMsg{Name: "sync", Text: "final output"})
	got := asModel(t, next)
	if got.watching != nil || got.output != "final output" {
		t.Errorf("commandMsg for the live watch left watching=%v output=%q", got.watching, got.output)
	}
	wantBand(t, got, "sync finished")

	// 2. A commandMsg for a watch that has already moved on: the output is
	// left alone.
	m.watching = &commandWatch{name: "other"}
	next, _ = m.Update(commandMsg{Name: "sync", Text: "stale output"})
	got = asModel(t, next)
	if got.watching == nil || got.output == "stale output" {
		t.Error("commandMsg for a stale watch touched state it should have left alone")
	}

	// 3. An error ends the run with the command's own words, verbatim.
	next, _ = m.Update(commandMsg{Name: "sync", Err: errors.New("exit status 1")})
	got = asModel(t, next)
	wantBand(t, got, "exit status 1")
}

func TestUpdateCliEndedMsg(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	next, _ := m.Update(cliEndedMsg{Engine: "claude", Repo: "payments"})
	got := asModel(t, next)
	if got.confirm != confirmPostCliTask || got.confirmID != "payments" {
		t.Errorf("cliEndedMsg with no error left confirm=%v confirmID=%q, want confirmPostCliTask/payments",
			got.confirm, got.confirmID)
	}
}

func TestUpdateEditorMsgBothBranches(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	next, cmd := m.Update(editorMsg{})
	got := asModel(t, next)
	if cmd != nil {
		t.Error("editorMsg with no error produced a command")
	}
	_ = got

	next, _ = m.Update(editorMsg{Err: errors.New("no such editor")})
	got = asModel(t, next)
	wantBand(t, got, "no such editor")
}

func TestUpdateDiffAndLogMsgDropStaleAnswers(t *testing.T) {
	m := openOn(t, "ACME-2662")

	// A diffMsg or logMsg for a task the reader has since left is dropped.
	next, cmd := m.Update(diffMsg{ID: "some-other-task", Text: "ignored"})
	got := asModel(t, next)
	if cmd != nil || got.diff == "ignored" {
		t.Error("diffMsg for a task the view is not open on was not dropped")
	}
	next, cmd = m.Update(logMsg{ID: "some-other-task"})
	got = asModel(t, next)
	if cmd != nil {
		t.Error("logMsg for a task the view is not open on was not dropped")
	}

	// The matching id lands.
	next, _ = m.Update(diffMsg{ID: "ACME-2662", Text: "the real diff"})
	got = asModel(t, next)
	wantPane(t, got, "the real diff")
}

// TestUpdateFallsThroughOnAnUnknownMessage is the one branch every case
// clause leaves behind: a message type Update was never told about answers
// with the model unchanged and no command, rather than a panic.
func TestUpdateFallsThroughOnAnUnknownMessage(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	type mysteryMsg struct{}
	next, cmd := m.Update(mysteryMsg{})
	if cmd != nil {
		t.Error("an unrecognised message produced a command")
	}
	if _, ok := next.(Model); !ok {
		t.Fatalf("Update returned %T for an unrecognised message, want ui.Model", next)
	}
}
