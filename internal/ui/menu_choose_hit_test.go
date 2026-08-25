package ui

// menu_more_coverage_test.go is the menu answering the two questions
// palette_and_menu_coverage_test.go's lifecycle test does not: what its
// entries are made of — a refused command greyed with its reason, an
// affordance dim with its own — and what choosing one of each actually
// does.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

func TestMenuEntriesForTheBoardShowRefusalsWhole(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = []Command{
		{Name: "new", About: func(p *words.Printer) string { return "write a task" }},
		{Name: "top", Refused: true, Because: func(p *words.Printer) string { return "you are already in it" }},
	}
	m = m.openMenu("")
	es := m.menuEntries()
	if len(es) != 2 {
		t.Fatalf("menuEntries() on the board menu = %d entries, want 2", len(es))
	}
	if es[0].dim || es[0].detail != "write a task" {
		t.Errorf("the runnable command entry is %+v, want its About shown and not dimmed", es[0])
	}
	if !es[1].dim || es[1].reason != "you are already in it" {
		t.Errorf("the refused command entry is %+v, want it dimmed with its reason", es[1])
	}

	// A task menu on an id still on the board lists its affordances.
	m2 := m.openMenu("ACME-2705")
	es2 := m2.menuEntries()
	if len(es2) == 0 {
		t.Fatal("menuEntries() on a live task answered with nothing")
	}

	// A task menu on an id that has left the board answers with nothing.
	m3 := m.openMenu("ACME-NOWHERE")
	if es3 := m3.menuEntries(); es3 != nil {
		t.Errorf("menuEntries() on a gone task = %v, want nil", es3)
	}
}

func TestOpenMenuForContextPicksTheRightSubject(t *testing.T) {
	// 1. The task view open on a subject opens the menu on it.
	m := openOn(t, "ACME-2705")
	after := m.openMenuForContext()
	if after.menu.taskID != "ACME-2705" {
		t.Errorf("openMenuForContext from the task view = %q, want ACME-2705", after.menu.taskID)
	}

	// 2. On the board, with a task under the cursor.
	m2, _ := testModel(t, 100, 30)
	m2 = onRow(t, m2, "ACME-2662")
	after2 := m2.openMenuForContext()
	if after2.menu.taskID != "ACME-2662" {
		t.Errorf("openMenuForContext on the board = %q, want ACME-2662", after2.menu.taskID)
	}

	// 3. On a band header, or with nothing selected: the board's own menu.
	m3, _ := testModel(t, 100, 30)
	m3.cursor = -1
	after3 := m3.openMenuForContext()
	if after3.menu.taskID != "" {
		t.Errorf("openMenuForContext with nothing selected = %q, want the board menu", after3.menu.taskID)
	}
}

func TestChooseMenuRunsACommandOrSendsAKey(t *testing.T) {
	// 1. Out of range: the menu is left exactly as it was.
	m, _ := testModel(t, 100, 30)
	m = m.openMenu("")
	m.menu.sel = 99
	next, cmd := m.chooseMenu()
	after := asModel(t, next)
	if cmd != nil || !after.menu.open || after.menu.sel != 99 {
		t.Errorf("chooseMenu out of range = open=%v sel=%v cmd=%v, want the menu untouched and nothing done", after.menu.open, after.menu.sel, cmd != nil)
	}

	// 2. A command entry runs through launch.
	m2, _ := testModel(t, 100, 30)
	m2.opts.Commands = []Command{{Name: "new"}}
	m2 = m2.openMenu("")
	m2.menu.sel = 0
	next2, _ := m2.chooseMenu()
	after2 := asModel(t, next2)
	if after2.menu.open {
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
	es := m3.menuEntries()
	pauseIdx := -1
	for i, e := range es {
		if e.aff != nil && e.glyph == "p" {
			pauseIdx = i
		}
	}
	if pauseIdx < 0 {
		t.Fatal("no pause entry on ACME-2705's menu")
	}
	m3.menu.sel = pauseIdx
	next3, cmd3 := m3.chooseMenu()
	after3 := asModel(t, next3)
	if after3.menu.open {
		t.Error("chooseMenu did not close the menu")
	}
	if cmd3 == nil {
		t.Fatal("choosing the pause entry answered with no command")
	}
	if _, ok := cmd3().(controlMsg); !ok {
		t.Errorf("choosing the pause entry raised %T, want a controlMsg", cmd3())
	}
}

func TestHitMenuMapsRowsToEntries(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = []Command{{Name: "new"}}
	m = m.openMenu("")

	// 1. A row past the frame's body answers nothing.
	if tgt := m.hitMenu(0, 9999); tgt.Kind != TargetNone {
		t.Errorf("hitMenu far past the body = %+v, want TargetNone", tgt)
	}

	// 2. A row on an entry answers a TargetMenuEntry keyed by its command.
	tgt := m.hitMenu(0, m.frame.Body.Y)
	if tgt.Kind != TargetMenuEntry || tgt.Key != "new" {
		t.Errorf("hitMenu on the first row = %+v, want TargetMenuEntry keyed \"new\"", tgt)
	}

	// 3. On a task menu, the row is keyed by the affordance's glyph.
	m2, _ := testModel(t, 100, 30)
	m2 = m2.openMenu("ACME-2705")
	tgt2 := m2.hitMenu(0, m2.frame.Body.Y)
	if tgt2.Kind != TargetMenuEntry || tgt2.Key == "" {
		t.Errorf("hitMenu on a task menu's first row = %+v, want a keyed TargetMenuEntry", tgt2)
	}
}

func TestMenuRowsAndMenuRowDrawWhatIsThere(t *testing.T) {
	// 1. h<=0 draws nothing.
	m, _ := testModel(t, 100, 30)
	if rows := m.menuRows(0, 40); rows != nil {
		t.Errorf("menuRows(0, ...) = %v, want nil", rows)
	}

	// 2. A menu whose task has left the board says so.
	gone := m.openMenu("ACME-NOWHERE")
	rows := gone.menuRows(6, 60)
	if !strings.Contains(strings.Join(rows, "\n"), "no longer on the board") {
		t.Errorf("menuRows on a gone task = %v, want it to say the task is gone", rows)
	}

	// 3. A normal menu draws one row per entry, selection marked.
	m.opts.Commands = []Command{{Name: "new"}, {Name: "top", Refused: true, Because: func(p *words.Printer) string { return "already here" }}}
	open := m.openMenu("")
	drawn := open.menuRows(10, 60)
	if len(drawn) != 10 {
		t.Errorf("menuRows(10, ...) drew %d lines, want 10 (padded)", len(drawn))
	}
	if !strings.Contains(drawn[0], "new") {
		t.Errorf("first menu row is %q, want it to mention \"new\"", drawn[0])
	}
}
