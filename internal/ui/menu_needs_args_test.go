package ui

// The one route between the menu and the command line.
//
// A command that refuses without arguments cannot be run from the board's
// menu: the menu chooses with none, and there is nowhere in it to say which
// task, or which directory. It used to run them bare, so choosing `requeue`
// printed "requeue needs the id of a task" into the watch — an answer to a
// question nobody asked, in a pane with nothing to type into.

import (
	"strings"
	"testing"
)

func needsArgsCommands() []Command {
	return []Command{
		{Name: "reconcile"},
		{Name: "pause", Args: "-repo <dir> <id>", NeedsArgs: true},
		{Name: "cancel", Args: "-repo <dir> <id>", NeedsArgs: true},
		{Name: "requeue", Args: "-repo <dir> <id> [why]", NeedsArgs: true},
		{Name: "export", Args: "[-task <id>] <dir>", NeedsArgs: true},
	}
}

// TestChoosingACommandThatNeedsArgumentsOpensTheLine: the three the reader
// reached for — pause, cancel, requeue — and export, which wants a
// directory rather than a task, all end at the line with the name already
// typed, and none of them runs.
func TestChoosingACommandThatNeedsArgumentsOpensTheLine(t *testing.T) {
	for _, name := range []string{"pause", "cancel", "requeue", "export"} {
		t.Run(name, func(t *testing.T) {
			m, _ := testModel(t, 100, 30)
			m.opts.Commands = needsArgsCommands()
			m = m.openMenu("")

			idx, ok := m.commandIndex(name)
			if !ok {
				t.Fatalf("no %s in the board's menu", name)
			}

			m.menu.sel = idx
			next, cmd := m.chooseMenu()

			after := asModel(t, next)
			if after.menu.open {
				t.Error("the menu stayed up")
			}

			if !after.palette.open || after.palette.typed != name+" " {
				t.Errorf("chose %s and got palette open=%v typed=%q, want the line up with the name on it",
					name, after.palette.open, after.palette.typed)
			}

			if cmd != nil {
				t.Errorf("choosing %s ran something: %T", name, cmd())
			}

			if after.watching != nil {
				t.Errorf("choosing %s opened the watch on %q", name, after.watching.name)
			}
		})
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

// TestTheLineRunsACommandOnceItHasItsArguments is the other half: with an
// id typed the command runs, and without one the line stays up and says
// what is missing rather than closing to print it somewhere else.
func TestTheLineRunsACommandOnceItHasItsArguments(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = needsArgsCommands()
	m.palette.open, m.palette.typed = true, "cancel "

	next, _ := m.runSelected()

	after := asModel(t, next)
	if !after.palette.open || after.palette.typed != "cancel " {
		t.Errorf("bare cancel left the line open=%v typed=%q, want it up and unchanged",
			after.palette.open, after.palette.typed)
	}

	if !strings.Contains(after.message, "cancel") {
		t.Errorf("bare cancel said %q, want it to name the command and ask for an id", after.message)
	}

	if after.watching != nil {
		t.Errorf("bare cancel ran %q", after.watching.name)
	}

	withID := m
	withID.palette.typed = "cancel PAY-11"

	afterID, _ := withID.runSelected()

	ran := asModel(t, afterID)
	if ran.palette.open {
		t.Error("the line stayed up after a command that ran")
	}

	if ran.watching == nil || ran.watching.name != "cancel" {
		t.Errorf("cancel PAY-11 left the watch on %v, want cancel running", ran.watching)
	}
}
