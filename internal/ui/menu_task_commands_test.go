package ui

// approve, permit and critical on the menu of the task they are about.
//
// They had no key and no letter anywhere in the window, so the command line
// was the only place they were listed — and that line is opened on the
// board, where there is no task for them to be about.

import (
	"slices"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// aTaskWithCommands is one task on the board and the table those three
// commands come from.
func aTaskWithCommands(t *testing.T) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{{ID: "ACME-7", Repo: "acme", RepoPath: "/checkouts/acme", Band: view.Done}}
	m.opts.Commands = []Command{
		{Name: "reconcile"},
		{Name: "approve", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true},
		{Name: "permit", Args: "-repo <dir> [-no] <id>", NeedsArgs: true, AboutATask: true},
		{Name: "critical", Args: "-repo <dir> [-off] <id>", NeedsArgs: true, AboutATask: true},
	}

	return m
}

// TestTheTaskMenuCarriesTheVerbsThatOnlyACommandDoes.
func TestTheTaskMenuCarriesTheVerbsThatOnlyACommandDoes(t *testing.T) {
	m := aTaskWithCommands(t).openMenu("ACME-7")

	var named []string

	for _, e := range m.menuEntries() {
		if e.cmd != nil {
			named = append(named, e.cmd.Name)
		}
	}

	if want := []string{"approve", "permit", "critical"}; !slices.Equal(named, want) {
		t.Errorf("the task's menu names %v, want %v", named, want)
	}
}

// TestTheyArriveKnowingWhichTask, which is the whole reason they moved: the
// menu was opened on a task, so the id and its repository are already known
// and the reader does not type them.
func TestTheyArriveKnowingWhichTask(t *testing.T) {
	m := aTaskWithCommands(t).openMenu("ACME-7")

	for _, e := range m.menuEntries() {
		if e.cmd == nil || e.cmd.Name != "permit" {
			continue
		}

		if want := []string{"-repo", "/checkouts/acme", "ACME-7"}; !slices.Equal(e.args, want) {
			t.Fatalf("permit is armed with %v, want %v", e.args, want)
		}

		return
	}

	t.Fatal("no permit on the menu of the task it is about")
}

// TestChoosingOneRunsItRatherThanAskingForArguments. A command that needs
// arguments and has none opens the line with its name on it; these have
// theirs, so they run, and the line — which no longer carries them — is
// never reached.
func TestChoosingOneRunsItRatherThanAskingForArguments(t *testing.T) {
	m := aTaskWithCommands(t).openMenu("ACME-7")
	m.menu.sel = menuIndex(t, m, "approve")

	next, cmd := m.chooseMenu()

	after := asModel(t, next)
	if after.palette.open {
		t.Error("approve went to the command line, which is the board's and has no task on it")
	}

	if after.watching == nil || after.watching.name != "approve" {
		t.Fatalf("choosing approve left the watch on %v, want approve running", after.watching)
	}

	if cmd == nil {
		t.Error("approve was watched with nothing to run")
	}
}

// TestTheBoardsMenuStillHasNoneOfThem: they are about one task, and the
// board's menu is opened on no row.
func TestTheBoardsMenuStillHasNoneOfThem(t *testing.T) {
	m := aTaskWithCommands(t).openMenu("")

	for _, e := range m.menuEntries() {
		if e.cmd != nil && e.cmd.AboutATask {
			t.Errorf("the board's menu offers %s, which is about one task", e.cmd.Name)
		}
	}
}

// TestATableWithoutThemDrawsNothingForThem. The window is handed its command
// table from outside; a build whose table has dropped one of these names has
// one row fewer, not a row naming a command that is not there.
func TestATableWithoutThemDrawsNothingForThem(t *testing.T) {
	m := aTaskWithCommands(t)
	m.opts.Commands = []Command{{Name: "reconcile"}}
	m = m.openMenu("ACME-7")

	for _, e := range m.menuEntries() {
		if e.cmd != nil {
			t.Errorf("the menu names %s, which this build's table does not have", e.cmd.Name)
		}
	}
}
