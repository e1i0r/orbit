package ui

// Two menus, two keys, and both of them answer a click.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/point"
)

// TestTheTwoMenusAreTwoKeys.
//
// m and M were one key: which menu it opened depended on whether the cursor
// happened to be on a row, so a reader could not tell in advance which they
// would get and pressed it to find out. m is the row's now and M is the
// board's, wherever the cursor is.
func TestTheTwoMenusAreTwoKeys(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2705")

	pressed, _ := m.Update(press(m.keys.Menu.Help().Key))

	onTask := asModel(t, pressed)
	if !onTask.menu.Up() {
		t.Fatal("m opened no menu")
	}

	if title := onTask.menu.Title(onTask.menuEnv()); !strings.Contains(title, "ACME-2705") {
		t.Errorf("m on a row opened %q, want the menu of the row", title)
	}

	// The board's, from the same row: what it is about is not this task.
	asked, _ := m.Update(press(m.keys.Board.Help().Key))

	onBoard := asModel(t, asked)
	if !onBoard.menu.Up() {
		t.Fatal("M opened no menu")
	}

	if title := onBoard.menu.Title(onBoard.menuEnv()); strings.Contains(title, "ACME-2705") {
		t.Errorf("M on a row opened %q, want the board's own menu", title)
	}
}

// TestBothMenusAnswerAClick. The bar carries both keys and a click on a
// hint is the keystroke: a key that can only be reached from the keyboard
// is half a key.
func TestBothMenusAnswerAClick(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2705")

	for _, c := range []struct {
		what string
		key  string
		task bool
	}{
		{"the task's menu", m.keys.Menu.Help().Key, true},
		{"the board's menu", m.keys.Board.Help().Key, false},
	} {
		clicked, _ := m.leftClick(point.Target{Kind: point.BarHint, Key: c.key})
		next := asModel(t, clicked)

		if !next.menu.Up() {
			t.Errorf("clicking %s opened nothing", c.what)

			continue
		}

		if about := strings.Contains(next.menu.Title(next.menuEnv()), "ACME-2705"); about != c.task {
			t.Errorf("clicking %s opened %q", c.what, next.menu.Title(next.menuEnv()))
		}
	}
}

// TestTheBoardsMenuIsOfferedOnlyWhereItWorks. A dialog takes every
// keystroke while it is up, so a bar offering M there would be the window
// promising something it does not do.
func TestTheBoardsMenuIsOfferedOnlyWhereItWorks(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2705")

	_, onList, _ := m.barLayout(m.frame.Bar.W)
	if !offers(onList, m.keys.Board.Help().Key) {
		t.Error("the board's own menu is not offered on the board")
	}

	inDialog, _ := dialog(t, m, "ACME-2662")

	_, hints, _ := inDialog.barLayout(inDialog.frame.Bar.W)
	if offers(hints, inDialog.keys.Board.Help().Key) {
		t.Error("the start dialog offers a key it does not answer")
	}
}

// offers is whether the bar is carrying a hint for that key.
func offers(hints []placedHint, key string) bool {
	for _, h := range hints {
		if h.key == key {
			return true
		}
	}

	return false
}
