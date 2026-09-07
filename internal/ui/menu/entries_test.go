package menu

// What the menu is a menu of.

import (
	"slices"
	"strings"
	"testing"
)

// TestTheBoardsMenuShowsRefusalsWhole. A refused command greyed with "you
// are already in it" teaches the shape of the program; one left off the
// list teaches nothing.
func TestTheBoardsMenuShowsRefusalsWhole(t *testing.T) {
	e := world(t)

	es := Open("", e).Entries(e)
	if len(es) != 3 {
		t.Fatalf("the board's menu is %d entries, want the three commands that are not about one task", len(es))
	}

	if es[0].Dim || es[0].Detail != "read the board again" {
		t.Errorf("the runnable command is %+v, want its description shown and not dimmed", es[0])
	}

	if !es[1].Dim || es[1].Reason != "you are already in it" {
		t.Errorf("the refused command is %+v, want it dimmed with its reason", es[1])
	}
}

// TestTheBoardsMenuHasNoVerbsAboutOneTask. It is opened on no row, so there
// is no task for such a verb to be about: choosing one ran the command bare
// and printed its usage into a pane with nothing to type into.
func TestTheBoardsMenuHasNoVerbsAboutOneTask(t *testing.T) {
	e := world(t)

	for _, entry := range Open("", e).Entries(e) {
		if entry.Command == "" {
			continue
		}

		for _, c := range e.Commands {
			if c.Name == entry.Command && c.AboutATask {
				t.Errorf("the board's menu offers %s, which is about one task", c.Name)
			}
		}
	}
}

// TestATaskThatLeftTheBoardHasNothingOnItsMenu, because every verb on it
// would be a verb about a run that is no longer there.
func TestATaskThatLeftTheBoardHasNothingOnItsMenu(t *testing.T) {
	e := world(t)

	if es := Open(gone, e).Entries(e); es != nil {
		t.Errorf("the menu of a task that left the board is %v, want nothing", es)
	}
}

// TestTheTaskMenuCarriesTheVerbsThatOnlyACommandDoes. Some of them had no
// key and no letter anywhere in the window, so the command line was the
// only place they were listed — and that line is opened on the board.
func TestTheTaskMenuCarriesTheVerbsThatOnlyACommandDoes(t *testing.T) {
	e := world(t)

	var named []string

	for _, entry := range Open(theTask, e).Entries(e) {
		if entry.Command != "" {
			named = append(named, entry.Command)
		}
	}

	want := []string{"note", "direct", "pr", "resolve", "merge", "close-pr", "approve", "permit", "critical"}
	if !slices.Equal(named, want) {
		t.Errorf("the task's menu names %v, want %v", named, want)
	}
}

// TestTheyArriveKnowingWhichTask, which is the whole reason they are here:
// the menu was opened on a task, so the id and its repository are already
// known and the reader does not type them.
func TestTheyArriveKnowingWhichTask(t *testing.T) {
	e := world(t)

	for _, entry := range Open(theTask, e).Entries(e) {
		if entry.Command != "permit" {
			continue
		}

		if want := []string{"-repo", "/checkouts/acme", theTask}; !slices.Equal(entry.Args, want) {
			t.Fatalf("permit is armed with %v, want %v", entry.Args, want)
		}

		return
	}

	t.Fatal("no permit on the menu of the task it is about")
}

// TestATableWithoutThemDrawsNothingForThem. The window is handed its
// command table from outside; a build whose table has dropped one of these
// names has one row fewer, not a row naming a command that is not there.
func TestATableWithoutThemDrawsNothingForThem(t *testing.T) {
	e := world(t)
	e.Commands = []Command{{Name: "reconcile"}}

	for _, entry := range Open(theTask, e).Entries(e) {
		if entry.Command != "" {
			t.Errorf("the menu names %s, which this build's table does not have", entry.Command)
		}
	}
}

// TestStartingARunIsOnTheMenuAsWell. `orbit run` is about a task like every
// other verb here, and the window's answer to it is a dialog rather than a
// command run bare — so the entry sends the key that opens the dialog.
func TestStartingARunIsOnTheMenuAsWell(t *testing.T) {
	e := world(t)

	want := e.Keys.Start.Help().Key
	for _, entry := range Open(theTask, e).Entries(e) {
		if entry.Glyph == want && entry.Command == "" && entry.Pane == "" {
			return
		}
	}

	t.Fatalf("nothing on the task's menu starts a run: %v", Open(theTask, e).Entries(e))
}

// TestARefusedVerbCarriesItsReason. A greyed entry with nothing beside it
// tells a reader they have done something wrong without telling them what.
func TestARefusedVerbCarriesItsReason(t *testing.T) {
	e := world(t)

	for _, entry := range Open(theTask, e).Entries(e) {
		if entry.Glyph != "c" {
			continue
		}

		if !entry.Dim || entry.Reason == "" {
			t.Fatalf("the refused verb is %+v, want it dimmed with the sentence saying why", entry)
		}

		return
	}

	t.Fatal("the refused verb is not on the menu at all")
}

// TestTheMenuInsideATaskOffersItsVerbsToo. Reading a run and acting on it
// happen in the same place, and the verbs used to be a screen back: a menu
// that offered eleven panes and nothing else said, by omission, that
// looking is all there is to do in here.
func TestTheMenuInsideATaskOffersItsVerbsToo(t *testing.T) {
	e := inside(t)

	var heads, shown, verbs int

	for _, entry := range Open(theTask, e).Entries(e) {
		switch {
		case entry.Head && entry.Title != "":
			heads++
		case entry.Pane != "":
			shown++
		case entry.Command == "":
			verbs++
		}
	}

	if heads != 2 {
		t.Errorf("the menu has %d headings, want one over the panes and one over the verbs", heads)
	}

	if shown != len(panes()) || verbs == 0 {
		t.Errorf("the menu has %d panes and %d verbs, want every pane and the task's verbs", shown, verbs)
	}
}

// TestInsideATaskThatLeftTheBoardThePanesAreStillThere. The panes are of
// the run being read, which is on the screen whatever the board says.
func TestInsideATaskThatLeftTheBoardThePanesAreStillThere(t *testing.T) {
	e := inside(t)

	es := Open(gone, e).Entries(e)
	if len(es) != len(panes())+1 {
		t.Fatalf("the menu is %d entries, want the heading and the twelve panes", len(es))
	}

	if !strings.Contains(es[0].Title, "panes") {
		t.Errorf("the first row is %q, want the heading the panes are listed under", es[0].Title)
	}
}
