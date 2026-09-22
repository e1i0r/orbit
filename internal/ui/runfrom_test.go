package ui

// The buttons on the flow tree: one per phase, mouse and keyboard.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/view"
)

// buttonGlyphs is every icon a node's button can be drawn under. The icon
// is the whole of what tells a press that starts something from a press
// that ends it, so a test looking for the button has to know all of them.
const buttonGlyphs = "▶↻■"

// buttonRow is the first row of lines carrying a node's button, or -1.
func buttonRow(lines []string) int {
	for i, l := range lines {
		if strings.ContainsAny(ansi.Strip(l), buttonGlyphs) {
			return i
		}
	}

	return -1
}

// onTheTree is a task screen with the flow tab up and its first node open.
func onTheTree(t *testing.T, open int) Model {
	t.Helper()

	m, _ := testModel(t, 120, 40)
	m.opts.Retry = func(view.Task, string, int) (int, error) { return 1, nil }

	id := ""

	for _, task := range m.board.Tasks {
		if view.BandOf(task) == view.NeedsYou {
			id = task.ID

			break
		}
	}

	if id == "" {
		t.Skip("no task on the fixture board to open a tree for")
	}

	next, _ := onto(t, m, id).openDetail(taskByID(t, m, id))
	m = asModel(t, next)
	m.tab = tabFlow

	if open >= 0 {
		m = m.openRow(open)
	}

	return m.syncPanes()
}

// TestAnOpenNodeOffersToRunFromThere.
//
// A run that stopped after implement left the reader with a tree saying
// review is pending and no way to say "then do review", short of a
// command line. `-from` already existed; there was nowhere to press it.
func TestAnOpenNodeOffersToRunFromThere(t *testing.T) {
	m := onTheTree(t, 0)

	rows, _ := m.flowRows()
	drawn := ansi.Strip(strings.Join(rows, "\n"))

	if !strings.ContainsAny(drawn, buttonGlyphs) {
		t.Errorf("an open node offers no button:\n%s", drawn)
	}
}

// TestAShutNodeOffersNothing: the button hangs off what a node says about
// itself, and a shut node says nothing.
func TestAShutNodeOffersNothing(t *testing.T) {
	m := onTheTree(t, -1)

	rows, _ := m.flowRows()
	if drawn := ansi.Strip(strings.Join(rows, "\n")); strings.ContainsAny(drawn, buttonGlyphs) {
		t.Errorf("a tree with every node shut still draws a button:\n%s", drawn)
	}
}

// TestTheButtonAnswersWhereItIsDrawn walks the pane row by row: the row
// that answers RunFrom is the row the ▶ was drawn on.
func TestTheButtonAnswersWhereItIsDrawn(t *testing.T) {
	m := onTheTree(t, 0)

	rows, _ := m.flowRows()

	drawnAt := buttonRow(rows)

	if drawnAt < 0 {
		t.Fatal("no button was drawn")
	}

	at, on := m.runFromAt(drawnAt)
	if !on {
		t.Fatalf("row %d draws the button and answers nothing", drawnAt)
	}

	if at != 0 {
		t.Errorf("the first phase's button answers phase %d", at)
	}

	// The rows around it are not the button.
	if _, on := m.runFromAt(drawnAt - 1); on {
		t.Error("the row above the button answers as the button")
	}
}

// TestAPhaseInFlightIsStoppedRatherThanStarted.
//
// The button says ■ on a phase that is running, and it means it: a second
// engine in the same worktree is the one mistake this cannot make, so the
// press stops the run instead. Signalled, because the control word is read
// at the next phase boundary and this phase has not reached one.
func TestAPhaseInFlightIsStoppedRatherThanStarted(t *testing.T) {
	m := onTheTree(t, 0)

	stopped := ""
	m.opts.Stop = func(task view.Task) error { stopped = task.ID; return nil }

	for i, task := range m.board.Tasks {
		if task.ID == m.detail {
			m.board.Tasks[i].Live = view.LiveHeld
		}
	}

	next, cmd := m.runFromPhase(0)
	if cmd == nil {
		t.Fatal("pressing ■ ran nothing")
	}

	cmd()

	if stopped != m.detail {
		t.Errorf("the run was not signalled; stopped = %q", stopped)
	}

	if got := asModel(t, next).await.verb; got != gestureCancel {
		t.Errorf("the window is awaiting %q, want %q", got, gestureCancel)
	}
}

// TestAHeldTaskWithNoStopPortIsRefusedOutLoud.
func TestAHeldTaskWithNoStopPortIsRefusedOutLoud(t *testing.T) {
	m := onTheTree(t, 0)
	m.opts.Stop = nil

	for i, task := range m.board.Tasks {
		if task.ID == m.detail {
			m.board.Tasks[i].Live = view.LiveHeld
		}
	}

	next, cmd := m.runFromPhase(0)
	if cmd != nil {
		t.Error("a held task was started again anyway")
	}

	if !strings.Contains(asModel(t, next).message, "running") {
		t.Errorf("it refused silently: %q", asModel(t, next).message)
	}
}

// TestTheButtonIsRouted, the guard both shipped bugs of this shape needed.
func TestTheButtonIsRouted(t *testing.T) {
	m := onTheTree(t, 0)

	after, run := m.leftClick(point.Target{Kind: point.RunFrom, Pane: 0})
	if run == nil {
		t.Fatal("a click on the button ran nothing")
	}

	if got := asModel(t, after).await.verb; got != gestureStart {
		t.Errorf("the click left the window awaiting %q, want %q", got, gestureStart)
	}
}

// TestAVerbStillOutIsLetGoOfByItsButton.
//
// The press on ■ ends the wait and writes down that it ended. What it does
// not do is pretend the verb was answered: the record keeps a cause saying
// nobody waited for it, which is what the tree then has under the node.
func TestAVerbStillOutIsLetGoOfByItsButton(t *testing.T) {
	m := onTheTree(t, 0)
	m.supervisorBusy = true
	m.delivering = deliverPending{task: taskByID(t, m, m.detail), verb: "RESOLVE COMMENTS", at: m.now}

	var wrote Delivery

	m.opts.RecordDeliver = func(_ view.Task, d Delivery) error { wrote = d; return nil }

	next, _ := m.stopWaiting("RESOLVE COMMENTS")
	after := asModel(t, next)

	if after.supervisorBusy {
		t.Error("the spinner is still turning on a verb nobody is waiting for")
	}

	if after.delivering.verb != "" {
		t.Errorf("the window is still waiting on %q", after.delivering.verb)
	}

	if !wrote.Done || wrote.Failure == nil {
		t.Errorf("the record does not say the verb was given up on: %+v", wrote)
	}
}

// TestAVerbThisWindowNeverAskedForIsNotStopped: stopping a wait nobody here
// is holding would write an ending onto work that is still running.
func TestAVerbThisWindowNeverAskedForIsNotStopped(t *testing.T) {
	m := onTheTree(t, 0)
	m.delivering = deliverPending{}

	wrote := false
	m.opts.RecordDeliver = func(view.Task, Delivery) error { wrote = true; return nil }

	next, _ := m.stopWaiting("RESOLVE COMMENTS")

	if wrote {
		t.Error("a verb asked for somewhere else was written down as ended here")
	}

	if got := asModel(t, next).message; !strings.Contains(got, "somewhere else") {
		t.Errorf("it refused silently: %q", got)
	}
}

// TestEveryDeliveryVerbCanBeAskedForAgain: a caption the button cannot
// route is a red node with a button on it that does nothing, which is the
// shape of bug this whole pass was about.
func TestEveryDeliveryVerbCanBeAskedForAgain(t *testing.T) {
	m := onTheTree(t, 0)

	for _, verb := range []string{
		"CREATE PR", "UPDATE PR", "FIX CHECKS", "MORE TESTS", "RESOLVE COMMENTS", "DEEP REVIEW",
	} {
		next, _ := m.askAgain(verb)
		if got := asModel(t, next).message; strings.Contains(got, "ask for again") {
			t.Errorf("%q has no gesture behind its button: %q", verb, got)
		}
	}
}

// taskByID is one task off the fixture board.
func taskByID(t *testing.T, m Model, id string) view.Task {
	t.Helper()

	for _, task := range m.board.Tasks {
		if task.ID == id {
			return task
		}
	}

	t.Fatalf("no task %s on the board", id)

	return view.Task{}
}
