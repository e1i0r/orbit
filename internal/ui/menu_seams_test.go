package ui

// The menu's two seams with the window: which subject it opens on, and what
// choosing a row actually reaches. What is on the menu and how it is drawn
// is the menu package's own suite.

import (
	"testing"
)

// pointAt puts the cursor on the entry with that glyph, the way a reader
// who walked down to it would: the menu is recomputed on every frame, so
// where a verb sits is not something a test can assume.
func pointAt(t *testing.T, m Model, glyph string) Model {
	t.Helper()

	env := m.menuEnv()
	for i, e := range m.menu.Entries(env) {
		if e.Glyph == glyph {
			m.menu = m.menu.Point(i)

			return m
		}
	}

	// Not on top: drill into each family and look there, the way a
	// reader who opened the menu would.
	for i, e := range m.menu.Entries(env) {
		if e.Family == "" {
			continue
		}

		sub, _ := m.menu.Point(i).Enter(env)
		for j, s := range sub.Entries(env) {
			if s.Glyph == glyph {
				m.menu = sub.Point(j)

				return m
			}
		}
	}

	t.Fatalf("no %q on the menu", glyph)

	return m
}

func TestOpenMenuForContextPicksTheRightSubject(t *testing.T) {
	// 1. The task view open on a subject opens the menu on it.
	m := openOn(t, "ACME-2705")

	after := m.openMenuForContext()
	if after.menu.Task() != "ACME-2705" {
		t.Errorf("openMenuForContext from the task view = %q, want ACME-2705", after.menu.Task())
	}

	// 2. On the board, with a task under the cursor.
	m2, _ := testModel(t, 100, 30)
	m2 = onRow(t, m2, "ACME-2662")

	after2 := m2.openMenuForContext()
	if after2.menu.Task() != "ACME-2662" {
		t.Errorf("openMenuForContext on the board = %q, want ACME-2662", after2.menu.Task())
	}

	// 3. On a band header, or with nothing selected: the board's own menu.
	m3, _ := testModel(t, 100, 30)
	m3.cursor = -1

	after3 := m3.openMenuForContext()
	if after3.menu.Task() != "" {
		t.Errorf("openMenuForContext with nothing selected = %q, want the board menu", after3.menu.Task())
	}
}

func TestChooseMenuRunsACommandOrSendsAKey(t *testing.T) {
	// 1. Out of range: the menu is left exactly as it was.
	m, _ := testModel(t, 100, 30)
	m = m.openMenu("")
	m.menu = m.menu.Point(99)
	next, cmd := m.chooseMenu()

	after := asModel(t, next)
	if cmd != nil || !after.menu.Up() || after.menu.At() != 99 {
		t.Errorf("chooseMenu out of range = open=%v sel=%v cmd=%v, want the menu untouched and nothing done",
			after.menu.Up(), after.menu.At(), cmd != nil)
	}

	// 2. A command entry runs through launch, which is where a command
	// that the window answers with a screen gets its screen.
	m2, _ := testModel(t, 100, 30)
	m2.opts.Commands = []Command{{Name: "new"}}
	m2 = m2.openMenu("")
	next2, _ := m2.chooseMenu()

	after2 := asModel(t, next2)
	if after2.menu.Up() {
		t.Error("chooseMenu did not close the menu")
	}

	if after2.screen != screenCompose {
		t.Errorf("chooseMenu on the new command left screen=%v, want screenCompose", after2.screen)
	}

	// 3. An affordance entry sends the keystroke its binding names — here
	// "p", offered on ACME-2705 the same way gesture_test.go's own case for
	// the key does, so the glyph and the binding's actual key agree.
	m3, _ := testModel(t, 100, 30)
	m3 = onRow(t, m3, "ACME-2705")
	m3 = m3.openMenu("ACME-2705")

	next3, cmd3 := pointAt(t, m3, "p").chooseMenu()

	after3 := asModel(t, next3)
	if after3.menu.Up() {
		t.Error("chooseMenu did not close the menu")
	}

	if cmd3 == nil {
		t.Fatal("choosing the pause entry answered with no command")
	}

	if _, ok := cmd3().(controlMsg); !ok {
		t.Errorf("choosing the pause entry raised %T, want a controlMsg", cmd3())
	}
}
