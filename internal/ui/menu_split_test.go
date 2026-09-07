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
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/palette"
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

// chooseInMenu puts the cursor on a named command and chooses it, the way a
// reader who walked down to it would. It is found by name and not by index:
// the menu leaves entries out and adds others, so where a command sits in
// the table is not where it sits here.
func chooseInMenu(t *testing.T, m Model, name string) (tea.Model, tea.Cmd) {
	t.Helper()

	for i, e := range m.menu.Entries(m.menuEnv()) {
		if e.Command != name {
			continue
		}

		m.menu = m.menu.Point(i)

		return m.chooseMenu()
	}

	t.Fatalf("no %s in the menu: %v", name, m.menu.Entries(m.menuEnv()))

	return m, nil
}

// TestChoosingACommandThatNeedsArgumentsOpensTheLine: export is generic —
// it wants a directory, not a task — so it stays on the board's menu, and
// choosing it ends at the line with the name already typed rather than
// running with nothing.
func TestChoosingACommandThatNeedsArgumentsOpensTheLine(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = needsArgsCommands()
	m = m.openMenu("")

	next, cmd := chooseInMenu(t, m, "export")

	after := asModel(t, next)
	if after.menu.Up() {
		t.Error("the menu stayed up")
	}

	if !after.palette.Up() || after.palette.Typed() != "export " {
		t.Errorf("chose export and got palette open=%v typed=%q, want the line up with the name on it",
			after.palette.Up(), after.palette.Typed())
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

	next, _ := chooseInMenu(t, m, "reconcile")

	after := asModel(t, next)
	if after.palette.Up() {
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

	next, _ := chooseInMenu(t, m, "new")

	after := asModel(t, next)
	if after.palette.Up() {
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
	m.palette = palette.OpenWith("export ")

	next, _ := m.paletteKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	after := asModel(t, next)
	if !after.palette.Up() || after.palette.Typed() != "export " {
		t.Errorf("bare export left the line open=%v typed=%q, want it up and unchanged",
			after.palette.Up(), after.palette.Typed())
	}

	if !strings.Contains(after.message, "export") {
		t.Errorf("bare export said %q, want it to name the command and ask for a directory", after.message)
	}

	if after.watching != nil {
		t.Errorf("bare export ran %q", after.watching.name)
	}

	withArgs := m
	withArgs.palette = palette.OpenWith("export /tmp/out")

	afterArgs, _ := withArgs.paletteKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	ran := asModel(t, afterArgs)
	if ran.palette.Up() {
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
	m.palette = palette.Open()

	// The list is read where the reader reads it: what the line drew.
	drawn := strings.Join(m.paletteRows(20, 100), "\n")

	for _, want := range []string{"reconcile", "export"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the line does not offer %q:\n%s", want, drawn)
		}
	}

	if strings.Contains(drawn, "cancel") {
		t.Errorf("the line offers a verb about one task:\n%s", drawn)
	}

	// And typing one says so, rather than leaving ⏎ on a row that is not
	// there.
	m.palette = palette.OpenWith("cancel")
	if got := strings.Join(m.paletteRows(20, 100), "\n"); !strings.Contains(got, "no command starts with") {
		t.Errorf("typing cancel drew:\n%s", got)
	}
}
