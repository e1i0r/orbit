package ui

// Walking the flow tree from the keyboard.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// caretAt is the row the cursor is drawn on, or -1.
func caretAt(t *testing.T, m Model) int {
	t.Helper()

	rows, _ := m.flowRows()
	for i, r := range rows {
		if strings.HasPrefix(ansi.Strip(r), treeCaret) {
			return i
		}
	}

	return -1
}

// TestTheTreeStartsWithACursorOnIt.
//
// A tree that showed no cursor until an arrow was pressed would be a tree
// whose first arrow does nothing a reader can see.
func TestTheTreeStartsWithACursorOnIt(t *testing.T) {
	if at := caretAt(t, onTheTree(t, -1)); at < 0 {
		t.Error("the tree opens with the keyboard standing nowhere")
	}
}

// TestTheArrowsWalkTheNodes: Elio, looking at a button he could see and
// could not reach — "no puedo ir con las flechitas".
func TestTheArrowsWalkTheNodes(t *testing.T) {
	m := onTheTree(t, -1)

	was := caretAt(t, m)

	next, _ := m.stepTree(1)
	m = asModel(t, next)

	down := caretAt(t, m)
	if down <= was {
		t.Fatalf("↓ left the cursor at row %d, was %d", down, was)
	}

	up, _ := m.stepTree(-1)
	if back := caretAt(t, asModel(t, up)); back != was {
		t.Errorf("↑ landed on row %d rather than back on %d", back, was)
	}
}

// TestTheCursorGoesRoundTheEnds: up from the first node is the last stop on
// the tree and down from there is the first again, as in every menu.
func TestTheCursorGoesRoundTheEnds(t *testing.T) {
	m := onTheTree(t, -1)
	first := caretAt(t, m)

	next, _ := m.stepTree(-1)
	m = asModel(t, next)

	if at := caretAt(t, m); at <= first {
		t.Fatalf("up from the first node left the caret on row %d, want the last stop below %d", at, first)
	}

	next, _ = m.stepTree(1)
	if at := caretAt(t, asModel(t, next)); at != first {
		t.Errorf("down from the last stop left the caret on row %d, want the first, %d", at, first)
	}
}

// TestEnterOpensTheNodeAndThenReachesItsButton.
//
// The whole of what the keyboard was missing: open a node, step onto what
// it offers, press it.
func TestEnterOpensTheNodeAndThenReachesItsButton(t *testing.T) {
	m := onTheTree(t, -1)

	opened, _ := m.pressTree()
	m = asModel(t, opened)

	if !m.rowOpen(tabFlow, m.tree.node) {
		t.Fatal("↵ on a shut node did not open it")
	}

	onButton, _ := m.stepTree(1)
	m = asModel(t, onButton)

	if !m.tree.button {
		t.Fatalf("↓ off an open node landed on %+v rather than on its button", m.tree)
	}

	if got := m.treeSaid(); got == "" {
		t.Error("the bar says nothing about what ↵ would do on a button")
	}

	started := ""
	m.opts.Retry = func(task view.Task, phase string, _ int) (int, error) {
		started = task.ID + " " + phase

		return 1, nil
	}

	next, cmd := m.pressTree()
	if cmd == nil {
		t.Fatal("↵ on the button ran nothing")
	}

	cmd()

	if started == "" {
		t.Error("↵ on the button did not start the phase")
	}

	if got := asModel(t, next).await.verb; got != gestureStart {
		t.Errorf("the window is awaiting %q, want %q", got, gestureStart)
	}
}

// TestTheBarSaysWhatEnterWouldDo: the word has to follow the cursor, since
// one key opens a node and presses a button.
func TestTheBarSaysWhatEnterWouldDo(t *testing.T) {
	m := onTheTree(t, -1)

	shut := m.treeSaid()

	opened, _ := m.pressTree()
	if got := asModel(t, opened).treeSaid(); got == shut {
		t.Errorf("the bar says %q over both a shut node and an open one", got)
	}
}

// TestAnotherTaskStartsTheCursorOver: a cursor kept across tasks would
// stand on a phase the new one does not walk.
func TestAnotherTaskStartsTheCursorOver(t *testing.T) {
	m := onTheTree(t, -1)

	moved, _ := m.stepTree(1)
	m = asModel(t, moved)

	if m.tree == (treeAt{}) {
		t.Skip("the fixture tree has one stop, so nothing moved")
	}

	next, _ := m.openDetail(taskByID(t, m, m.detail))
	if next.tree != (treeAt{}) {
		t.Errorf("opening a task left the cursor at %+v", next.tree)
	}
}
