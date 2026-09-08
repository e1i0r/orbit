//go:build integration

package cli

// The six flows the landing shows that are about the board rather than about
// a run: the task that was just written down, the menu, reading a task, the
// supervisor, what Orbit knows, and the flows screen.
//
// One test per recording, pressing what the recording presses.
//
// Every one of them is named TestCockpit… because that prefix is what
// `make integration` selects on: a test added here under another name is a
// test nothing runs.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestCockpitShowsTheTaskJustWrittenDown — flow-start: `orbit new`, then
// `orbit top`, and the row is there.
func TestCockpitShowsTheTaskJustWrittenDown(t *testing.T) {
	ctx, dir := cockpitBoard(t)

	m := opened(t, ctx, dir)

	// The window opens with To Do shut — it answers "what needs me" — and
	// the cursor starts on that band's own head, where enter opens it.
	frame := drawn(t, press(t, m, named(tea.KeyEnter)))

	for _, want := range []string{"LED-1", "refund"} {
		if !strings.Contains(frame, want) {
			t.Errorf("the board does not show %q:\n%s", want, frame)
		}
	}
}

// TestCockpitMenuOpensAndCloses — flow-menus: m opens it, escape puts it away,
// and the board is underneath it again.
func TestCockpitMenuOpensAndCloses(t *testing.T) {
	ctx, dir := cockpitBoard(t)

	m := opened(t, ctx, dir)
	board := drawn(t, m)

	opened := drawn(t, press(t, m, typed("m")))
	if opened == board {
		t.Fatalf("m drew the same frame as the board:\n%s", opened)
	}

	shut := drawn(t, press(t, m, typed("m"), named(tea.KeyEscape)))
	if shut != board {
		t.Errorf("escape did not put the menu away:\n%s\n\nwant:\n%s", shut, board)
	}
}

// TestCockpitOpensATaskOnItsPanes — flow-reading: enter on a row opens the
// task, and the numbered keys walk its panes.
func TestCockpitOpensATaskOnItsPanes(t *testing.T) {
	ctx, dir := cockpitBoard(t)

	m := opened(t, ctx, dir)
	board := drawn(t, m)

	// Enter on the band's head opens the band, down steps onto the row, and
	// enter there opens the task.
	onTask := press(t, m, named(tea.KeyEnter), named(tea.KeyDown), named(tea.KeyEnter))

	task := drawn(t, onTask)
	if task == board {
		t.Fatalf("enter did not open the task:\n%s", task)
	}

	if !strings.Contains(task, "LED-1") {
		t.Errorf("the task that opened is not the one the cursor was on:\n%s", task)
	}

	report := drawn(t, press(t, onTask, typed("7")))
	if report == task {
		t.Errorf("7 did not move to another pane:\n%s", report)
	}
}

// TestCockpitOpensTheSupervisor — flow-supervisor: S, and the thread is there to be
// written into.
func TestCockpitOpensTheSupervisor(t *testing.T) {
	ctx, dir := cockpitBoard(t)

	m := opened(t, ctx, dir)

	if got := drawn(t, press(t, m, typed("S"))); got == drawn(t, m) {
		t.Errorf("S did not open the supervisor:\n%s", got)
	}
}

// TestCockpitOpensWhatOrbitKnows — flow-knowledge: K.
func TestCockpitOpensWhatOrbitKnows(t *testing.T) {
	ctx, dir := cockpitBoard(t)

	m := opened(t, ctx, dir)

	if got := drawn(t, press(t, m, typed("K"))); got == drawn(t, m) {
		t.Errorf("K did not open what orbit knows:\n%s", got)
	}
}

// TestCockpitFlowsScreenNamesTheFlowsThatShip — flow-flows: F, and the five that
// ship are named on it.
func TestCockpitFlowsScreenNamesTheFlowsThatShip(t *testing.T) {
	ctx, dir := cockpitBoard(t)

	m := opened(t, ctx, dir)
	flows := drawn(t, press(t, m, typed("F")))

	for _, want := range []string{"quick", "task", "careful", "coverage", "tdd-fuzz-pr"} {
		if !strings.Contains(flows, want) {
			t.Errorf("the flows screen does not name %q:\n%s", want, flows)
		}
	}
}
