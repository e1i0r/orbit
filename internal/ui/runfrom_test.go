package ui

// The buttons on the flow tree: one per phase, mouse and keyboard.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/view"
)

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

	if !strings.Contains(drawn, "▶") {
		t.Errorf("an open node offers no button:\n%s", drawn)
	}

	// And it names the key, so the gesture explains itself.
	if !strings.Contains(drawn, m.keys.RetryPhase.Help().Key) {
		t.Errorf("the button does not name the key that presses it:\n%s", drawn)
	}
}

// TestAShutNodeOffersNothing: the button hangs off what a node says about
// itself, and a shut node says nothing.
func TestAShutNodeOffersNothing(t *testing.T) {
	m := onTheTree(t, -1)

	rows, _ := m.flowRows()
	if drawn := ansi.Strip(strings.Join(rows, "\n")); strings.Contains(drawn, "▶") {
		t.Errorf("a tree with every node shut still draws a button:\n%s", drawn)
	}
}

// TestTheButtonAnswersWhereItIsDrawn walks the pane row by row: the row
// that answers RunFrom is the row the ▶ was drawn on.
func TestTheButtonAnswersWhereItIsDrawn(t *testing.T) {
	m := onTheTree(t, 0)

	rows, _ := m.flowRows()

	drawnAt := -1

	for i, r := range rows {
		if strings.Contains(ansi.Strip(r), "▶") {
			drawnAt = i

			break
		}
	}

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

// TestTheKeyPressesTheOpenNodesButton is the keyboard half. Opening a node
// is how a reader says which phase they mean, and ^R presses that one.
func TestTheKeyPressesTheOpenNodesButton(t *testing.T) {
	m := onTheTree(t, 1)

	phase, aiming := m.aimed()
	if !aiming {
		t.Fatal("one open node and the key has nothing to aim at")
	}

	if phase == "" {
		t.Error("the open node names no phase")
	}

	// Two open nodes is two answers, and picking either would be the
	// window choosing for the reader.
	two := m.openRow(0)
	if _, aiming := two.aimed(); aiming {
		t.Error("two open nodes still aimed at one")
	}

	// None open falls back to the phase that went wrong, which is what
	// the key did before there were buttons.
	none := onTheTree(t, -1)
	if _, aiming := none.aimed(); aiming {
		t.Error("no open node still aimed at one")
	}
}

// TestRunningFromAPhaseIsRefusedWhileTheTaskIsHeld: a second engine in the
// same worktree is the one mistake this cannot make.
func TestRunningFromAPhaseIsRefusedWhileTheTaskIsHeld(t *testing.T) {
	m := onTheTree(t, 0)

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
