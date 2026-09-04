package ui

// The two menus and the one route between the board's and the command line.
//
// The menu m opens is the menu of whatever the pointer was on. On a row it
// is that task's verbs; on a band head, or on nothing, it is the board's —
// and a verb about one task has no place there, because there is no task
// for it to be about. Choosing `requeue` from it ran the command bare and
// printed "requeue needs the id of a task" into the watch: an answer to a
// question nobody asked, in a pane with nothing to type into.
//
// What is left on the board's menu can still want an argument — `export`
// wants a directory, which is not a task — and that is the route to the
// line, with the name already typed.

import (
	"slices"
	"strings"
	"testing"
)

func needsArgsCommands() []Command {
	return []Command{
		{Name: "reconcile"},
		{Name: "pause", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true},
		{Name: "cancel", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true},
		{Name: "requeue", Args: "-repo <dir> <id> [why]", NeedsArgs: true, AboutATask: true},
		{Name: "export", Args: "[-task <id>] <dir>", NeedsArgs: true},
	}
}

// menuIndex is where a named command sits in the menu as it is drawn, which
// is no longer where it sits in the table: the board's menu leaves entries
// out, so an index into one is not an index into the other.
func menuIndex(t *testing.T, m Model, name string) int {
	t.Helper()

	for i, e := range m.menuEntries() {
		if e.cmd != nil && e.cmd.Name == name {
			return i
		}
	}

	t.Fatalf("no %s in the menu: %v", name, m.menuEntries())

	return 0
}

// The three verbs the reader reached for are not on the board's menu at
// all. This is the fix and not the routing: an entry that sends the reader
// somewhere else to say which task is an entry in the wrong menu.
func TestTheBoardsMenuHasNoVerbsAboutOneTask(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = needsArgsCommands()
	m = m.openMenu("")

	for _, e := range m.menuEntries() {
		if e.cmd != nil && e.cmd.AboutATask {
			t.Errorf("the board's menu offers %s, which is about one task", e.cmd.Name)
		}
	}

	// And what is generic is still there, wanting an argument or not.
	menuIndex(t, m, "reconcile")
	menuIndex(t, m, "export")
}

// TestChoosingACommandThatNeedsArgumentsOpensTheLine: export is generic —
// it wants a directory, not a task — so it stays on the board's menu, and
// choosing it ends at the line with the name already typed rather than
// running with nothing.
func TestChoosingACommandThatNeedsArgumentsOpensTheLine(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = needsArgsCommands()
	m = m.openMenu("")
	m.menu.sel = menuIndex(t, m, "export")

	next, cmd := m.chooseMenu()

	after := asModel(t, next)
	if after.menu.open {
		t.Error("the menu stayed up")
	}

	if !after.palette.open || after.palette.typed != "export " {
		t.Errorf("chose export and got palette open=%v typed=%q, want the line up with the name on it",
			after.palette.open, after.palette.typed)
	}

	if cmd != nil {
		t.Errorf("choosing export ran something: %T", cmd())
	}

	if after.watching != nil {
		t.Errorf("choosing export opened the watch on %q", after.watching.name)
	}
}

// A command that wants nothing is untouched by any of this: it still runs
// the moment it is chosen.
func TestChoosingACommandThatNeedsNothingStillRuns(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = needsArgsCommands()
	m = m.openMenu("")
	m.menu.sel = 0

	next, _ := m.chooseMenu()

	after := asModel(t, next)
	if after.palette.open {
		t.Error("reconcile went to the command line, want it run")
	}

	if after.watching == nil || after.watching.name != "reconcile" {
		t.Errorf("chose reconcile and the watch is %v, want it watching reconcile", after.watching)
	}
}

// A command the window answers with a screen keeps its screen, whatever the
// table says it wants: `new` refuses on the command line without -id, and
// the compose form is where that id is filled in. The order these two rules
// are applied in is the whole of this test.
func TestACommandWithAScreenKeepsItOverTheCommandLine(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = []Command{{Name: "new", Args: "-repo <dir> -id <id> <text>", NeedsArgs: true}}
	m = m.openMenu("")
	m.menu.sel = 0

	next, _ := m.chooseMenu()

	after := asModel(t, next)
	if after.palette.open {
		t.Error("new went to the command line, want the compose screen")
	}

	if after.screen != screenCompose {
		t.Errorf("chose new and the screen is %v, want screenCompose", after.screen)
	}
}

// TestTheLineRunsACommandOnceItHasItsArguments is the other half: with what
// it needs typed the command runs, and without it the line stays up and says
// what is missing rather than closing to print it somewhere else.
//
// The command is export and not cancel because cancel is about one task and
// the line no longer carries those. What is being tested is the line's rule
// about arguments, which export needs a directory for.
func TestTheLineRunsACommandOnceItHasItsArguments(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = needsArgsCommands()
	m.palette.open, m.palette.typed = true, "export "

	next, _ := m.runSelected()

	after := asModel(t, next)
	if !after.palette.open || after.palette.typed != "export " {
		t.Errorf("bare export left the line open=%v typed=%q, want it up and unchanged",
			after.palette.open, after.palette.typed)
	}

	if !strings.Contains(after.message, "export") {
		t.Errorf("bare export said %q, want it to name the command and ask for a directory", after.message)
	}

	if after.watching != nil {
		t.Errorf("bare export ran %q", after.watching.name)
	}

	withArgs := m
	withArgs.palette.typed = "export /tmp/out"

	afterArgs, _ := withArgs.runSelected()

	ran := asModel(t, afterArgs)
	if ran.palette.open {
		t.Error("the line stayed up after a command that ran")
	}

	if ran.watching == nil || ran.watching.name != "export" {
		t.Errorf("export /tmp/out left the watch on %v, want export running", ran.watching)
	}
}

// TestTheLineLeavesOutTheVerbsAboutOneTask, for the reason the board's menu
// does: the line is opened on the board, where there is no task for such a
// verb to be about. It offered them and then answered with a usage string
// to satisfy by hand — for a task that was on the screen behind it.
func TestTheLineLeavesOutTheVerbsAboutOneTask(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = needsArgsCommands()
	m.palette.open = true

	var names []string
	for _, c := range m.palette.candidates(m.opts.Commands) {
		names = append(names, c.Name)
	}

	if want := []string{"reconcile", "export"}; !slices.Equal(names, want) {
		t.Errorf("the line offers %v, want %v", names, want)
	}

	// And typing one says so, rather than leaving ⏎ on a row that is not
	// there.
	m.palette.typed = "cancel"
	if n := len(m.palette.candidates(m.opts.Commands)); n != 0 {
		t.Errorf("typing cancel matched %d commands, want none", n)
	}
}

// Each menu says which one it is, in a line above the entries. The reader
// who opened the board's meaning the task's should be able to see that from
// the menu rather than from a verb that is not in it.
func TestEachMenuSaysWhichOneItIs(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = needsArgsCommands()

	board := m.openMenu("").menuTitle()
	if !strings.Contains(board, "one task") {
		t.Errorf("the board's menu is titled %q, want it to say it is about no one task", board)
	}

	task := m.openMenu("ACME-2705").menuTitle()
	if !strings.Contains(task, "ACME-2705") {
		t.Errorf("a task's menu is titled %q, want it to name the task", task)
	}

	// And the title is drawn, not merely computed.
	rows := strings.Join(m.openMenu("").menuRows(20, 100), "\n")
	if !strings.Contains(rows, board) {
		t.Errorf("the menu drew %q, want the title in it", rows)
	}
}
