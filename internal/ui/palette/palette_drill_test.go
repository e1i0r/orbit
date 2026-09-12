package palette

// Drilling into a family from the line: naming a parent lists it with its
// children, choosing it goes one level down, and choosing a child runs
// the parent with the whole line.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// family is one parent with two children, the way the command table hands
// families over.
func family() []Command {
	return []Command{
		{Name: "pr", Args: "<id>", About: "a pull request", Children: []Child{
			{Name: "merge", About: "merge it"},
			{Name: "close", About: "close it"},
		}},
	}
}

// world is the Env: the words, the keys, a body of five rows, and that
// table.

// TestNamingAParentListsItWithItsChildren. The parent stands at the head
// of them, so that choosing it drills in and choosing one of them runs it.
func TestNamingAParentListsItWithItsChildren(t *testing.T) {
	e := world(t)
	e.Commands = family()

	var got []string

	for _, c := range OpenWith("pr").candidates(e.Commands) {
		got = append(got, c.Name)
	}

	want := []string{"pr", "pr merge", "pr close"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("naming pr listed %v, want %v", got, want)
	}
}

// TestASecondWordFiltersTheChildren, the way the first filters the
// commands: what follows the parent is the family's to read.
func TestASecondWordFiltersTheChildren(t *testing.T) {
	e := world(t)
	e.Commands = family()

	var got []string

	for _, c := range OpenWith("pr m").candidates(e.Commands) {
		got = append(got, c.Name)
	}

	if len(got) != 1 || got[0] != "pr merge" {
		t.Errorf("pr m listed %v, want only pr merge", got)
	}
}

// TestChoosingAParentDrillsInsteadOfRunning. Nothing is run with half a
// name: the line keeps the parent and the list becomes its children.
func TestChoosingAParentDrillsInsteadOfRunning(t *testing.T) {
	e := world(t)
	e.Commands = family()

	next, out := OpenWith("pr").Run(e)
	if out.Run != "" || !next.Up() {
		t.Errorf("choosing pr answered run=%q with the line up=%v", out.Run, next.Up())
	}

	if next.Typed() != "pr " {
		t.Errorf("the line reads %q, want the parent with room for the child", next.Typed())
	}

	var got []string

	for _, c := range next.candidates(e.Commands) {
		got = append(got, c.Name)
	}

	if strings.Join(got, ",") != "pr,pr merge,pr close" {
		t.Errorf("after drilling the list is %v, want the parent over the children", got)
	}
}

// TestChoosingAChildRunsTheParentWithTheWholeLine. Every door runs a child
// through its parent, and the line carries both words plus the arguments.
func TestChoosingAChildRunsTheParentWithTheWholeLine(t *testing.T) {
	e := world(t)
	e.Commands = family()

	next, out := OpenWith("pr merge").Run(e)
	if out.Run != "pr" || !out.Leave || next.Up() {
		t.Errorf("choosing pr merge answered %+v with the line up=%v", out, next.Up())
	}

	if out.Line != "pr merge" {
		t.Errorf("the line handed over is %q, want both words", out.Line)
	}
}

// TestTabOnAParentLeavesRoomForTheChild, so the list becomes the family
// rather than one row racing the next keystroke.
func TestTabOnAParentLeavesRoomForTheChild(t *testing.T) {
	e := world(t)
	e.Commands = family()

	after, _ := OpenWith("pr").Key(tea.KeyPressMsg{Code: tea.KeyTab}, e)
	if after.Typed() != "pr " {
		t.Errorf("tab left %q on the line, want the parent with room", after.Typed())
	}
}
