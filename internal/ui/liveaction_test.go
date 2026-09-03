package ui

// The live action of a running task, drawn in two places that have very
// different amounts of room for it, and the measure each of them cuts it to.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// aCall is one tool call as view.ToolLine hands it over: longer than the
// fifty characters the band has room for, shorter than the pane.
const aCall = "Bash: go test ./internal/task/ -run TestTheShippedTaskFlow -count=1 -v"

// liveAction puts one tool call on a task of the fixture board, as a
// phase.tool_call event would.
func liveAction(m *Model, id, action string) {
	for i, t := range m.board.Tasks {
		if t.ID == id {
			m.board.Tasks[i].CurrentAction, m.board.Tasks[i].ActionKind = action, view.ActionTool
		}
	}
}

// TestTheOverviewDrawsTheCallItHasRoomFor.
//
// The action was cut to fifty characters in internal/view, where the band's
// row is what fifty is a measure of. The overview draws it on a line it
// shares with nothing, so the reader saw `tail -20 /tmp/fra62-check4.log
// 2>/dev/null; echo …` with eighty columns of empty pane beside it.
func TestTheOverviewDrawsTheCallItHasRoomFor(t *testing.T) {
	m := openOn(t, "ACME-2705")
	liveAction(&m, "ACME-2705", aCall)

	if got := overviewText(m); !strings.Contains(got, aCall) {
		t.Errorf("the overview cut a call the pane had room for:\n%s", got)
	}
}

// TestTheOverviewStillCutsWhatThePaneCannotHold. Cut to the pane is not the
// same as not cut: a call wider than the window is still one row, and the
// ellipsis is how the reader knows the row was cut.
func TestTheOverviewStillCutsWhatThePaneCannotHold(t *testing.T) {
	m := openOn(t, "ACME-2705")
	wider := "Bash: " + strings.Repeat("ñ", m.frame.Body.W+20)
	liveAction(&m, "ACME-2705", wider)

	got := overviewText(m)
	if !strings.Contains(got, "…") {
		t.Errorf("a call wider than the pane was not marked as cut:\n%s", got)
	}

	for _, line := range strings.Split(got, "\n") {
		if w := ansi.StringWidth(line); w > m.frame.Body.W {
			t.Errorf("the overview drew a line %d cells wide in a pane of %d: %q", w, m.frame.Body.W, line)
		}
	}
}

// TestTheBandCutsTheCallToWhatTheRowHasLeft. The band's row carries the id,
// the phase, the elapsed time, the engine and the flow as well, and the
// whole of it is cut to the terminal at the end: an action left whole is an
// action that pushes the engine off the end.
func TestTheBandCutsTheCallToWhatTheRowHasLeft(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	got := ansi.Strip(m.runningLine(view.Task{
		ID: "ACME-2705", Band: view.Running, Phase: "implement",
		Engine: "claude", Model: "opus", Flow: "careful",
		CurrentAction: aCall, ActionKind: view.ActionTool,
	}))

	if strings.Contains(got, aCall) {
		t.Errorf("the band drew the whole call: %q", got)
	}

	if !strings.Contains(got, "Bash: go test ./internal/task/") {
		t.Errorf("the band = %q, want it to name what the call is about", got)
	}

	for _, want := range []string{"claude/opus", "careful"} {
		if !strings.Contains(got, want) {
			t.Errorf("the band = %q, want %q still on the row after the action", got, want)
		}
	}
}
