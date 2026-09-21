package repos

// The repository list: what it collects, and every key it answers.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// worked is a board of three tasks: two filed under repositories the reader
// named, and one that reaches into a third nobody listed.
func worked3() board.Board {
	return board.Board{
		RepoList: []board.RepoInfo{
			{Name: "payments", Path: "/r/payments"},
			{Name: "app", Path: "/r/app"},
		},
		Tasks: []view.Task{
			{Repo: "payments", ID: "ACME-1", Band: view.NeedsYou},
			{Repo: "app", ID: "ACME-2", Band: view.Running},
			{Repo: "app", Repos: []string{"app", "api"}, ID: "ACME-3", Band: view.Running},
		},
	}
}

// world is the Env: the words, the keys, and that board.
func world(t *testing.T) Env {
	t.Helper()

	return Env{
		Words: words.For("en"),
		Keys:  keymap.New(words.For("en")),
		Board: worked3(),
	}
}

// press is one keystroke as the event loop delivers it.
func press(keystroke string) tea.KeyPressMsg {
	switch keystroke {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "space":
		return tea.KeyPressMsg{Code: ' ', Text: " "}
	}

	r := []rune(keystroke)[0]

	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// TestARepositoryATaskReachesIntoIsStillOnTheList. A task that reached into
// three checkouts is three repositories with work going on in them, and a
// list that left out the one nobody opened the board over would be a list of
// repositories missing one that work is happening in.
func TestARepositoryATaskReachesIntoIsStillOnTheList(t *testing.T) {
	got := List(world(t))
	if len(got) != 3 {
		t.Fatalf("the list has %d repositories, want the two named and the one a task reaches into", len(got))
	}

	var named []string

	total := 0

	for _, r := range got {
		named = append(named, r.Name)
		total += r.total
	}

	if total == 0 {
		t.Errorf("no task counted against any repository: %v", named)
	}

	if !strings.Contains(strings.Join(named, " "), "api") {
		t.Errorf("the list is %v, want the checkout ACME-3 reaches into", named)
	}
}

// TestEveryBandCountsAgainstTheRepositoryItIsIn. The four numbers beside a
// repository are what the reader chooses one by, and the band a task is in
// is what decides which of them moves: a band left out counts nowhere, and
// the row then says a repository has less going on in it than it has.
func TestEveryBandCountsAgainstTheRepositoryItIsIn(t *testing.T) {
	e := world(t)
	e.Board = board.Board{
		RepoList: []board.RepoInfo{{Name: "payments", Path: "/r/payments"}},
		Tasks: []view.Task{
			{Repo: "payments", ID: "ACME-1", Band: view.ToDo},
			{Repo: "payments", ID: "ACME-2", Band: view.Running},
			{Repo: "payments", ID: "ACME-3", Band: view.NeedsYou},
			{Repo: "payments", ID: "ACME-4", Band: view.Done},
		},
	}

	got := List(e)
	if len(got) != 1 {
		t.Fatalf("one repository with four tasks came back as %d rows", len(got))
	}

	for band, want := range map[view.Band]int{
		view.ToDo: 1, view.NeedsYou: 1, view.Running: 1, view.Done: 1,
	} {
		if n := got[0].counts[band]; n != want {
			t.Errorf("the %s column counts %d, want %d", band, n, want)
		}
	}

	if got[0].total != 4 {
		t.Errorf("four tasks in one repository total %d", got[0].total)
	}
}

// TestARowWithNoListOfCheckoutsCountsUnderTheOneItIsFiledIn. Most rows carry
// no list at all — one task, one repository — and a row read as reaching
// into nothing is a task counted against no repository on a screen whose
// whole subject is which repository the work is in.
func TestARowWithNoListOfCheckoutsCountsUnderTheOneItIsFiledIn(t *testing.T) {
	e := world(t)
	e.Board = board.Board{Tasks: []view.Task{{Repo: "payments", ID: "ACME-1", Band: view.Running}}}

	got := List(e)
	if len(got) != 1 || got[0].Name != "payments" {
		t.Fatalf("a task filed under payments and reaching nowhere listed %+v", got)
	}

	if got[0].total != 1 {
		t.Errorf("it counts %d tasks against payments, want 1", got[0].total)
	}
}

// TestWithNoListTheRepositoriesComeFromTheTasks, once each.
func TestWithNoListTheRepositoriesComeFromTheTasks(t *testing.T) {
	e := world(t)
	e.Board.RepoList = nil

	got := List(e)
	if len(got) == 0 {
		t.Fatal("a board with no list of its own found no repositories in its tasks")
	}

	seen := map[string]bool{}
	for _, r := range got {
		if seen[strings.ToLower(r.Name)] {
			t.Errorf("%q is on the list twice", r.Name)
		}

		seen[strings.ToLower(r.Name)] = true
	}
}

// TestTheArrowsWrapAtBothEnds.
func TestTheArrowsWrapAtBothEnds(t *testing.T) {
	e := world(t)
	s := Open()

	last := len(List(e)) - 1

	up, _ := s.Key(press("up"), e)
	if up.sel != last {
		t.Errorf("up from the first row = %d, want %d", up.sel, last)
	}

	if down, _ := up.Key(press("down"), e); down.sel != 0 {
		t.Errorf("down from the last row = %d, want 0", down.sel)
	}

	// And the row above the second one is the first, not the last: the
	// wrap is the end of the list and not every step through it.
	second, _ := Open().Key(press("down"), e)

	if back, _ := second.Key(press("up"), e); back.sel != 0 {
		t.Errorf("up from the second row = %d, want the first", back.sel)
	}
}

// TestChoosingFiltersAndChoosingAgainClearsIt. The same gesture both ways:
// a reader who filtered by pressing ⏎ should not have to find another key to
// undo it.
func TestChoosingFiltersAndChoosingAgainClearsIt(t *testing.T) {
	e := world(t)
	first := List(e)[0].Name

	_, out := Open().Key(press("enter"), e)
	if !out.Leave || out.Filter != first {
		t.Fatalf("⏎ on the first row answered leave=%v filter=%q, want %q", out.Leave, out.Filter, first)
	}

	if out.Said == "" {
		t.Error("filtering the board said nothing about it")
	}

	// Now the board is showing it, and the same key takes the filter off.
	e.Filter = first

	_, out = Open().Key(press("enter"), e)
	if !out.Cleared || out.Filter != "" {
		t.Errorf("⏎ on the repository already showing answered cleared=%v filter=%q",
			out.Cleared, out.Filter)
	}
}

// TestSpaceDoesWhatEnterDoes, because a list is a thing people press space
// at.
func TestSpaceDoesWhatEnterDoes(t *testing.T) {
	e := world(t)

	if _, out := Open().Key(press("space"), e); out.Filter == "" {
		t.Error("space on a repository row did not filter to it")
	}
}

// TestAClickChoosesTheRepositoryItLandedOn, which is the same answer the
// keyboard gives.
func TestAClickChoosesTheRepositoryItLandedOn(t *testing.T) {
	e := world(t)

	if _, out := Open().Choose("app", e); out.Filter != "app" {
		t.Errorf("clicking app answered filter=%q", out.Filter)
	}

	// A name nothing on the board carries changes nothing rather than
	// filtering to a repository that is not there.
	if _, out := Open().Choose("nowhere", e); out.Leave || out.Filter != "" {
		t.Errorf("clicking a repository that is not listed answered %+v", out)
	}
}

// TestWithNothingToListOnlyLeavingWorks.
func TestWithNothingToListOnlyLeavingWorks(t *testing.T) {
	e := world(t)
	e.Board = board.Board{}

	if _, out := Open().Key(press("down"), e); out.Leave {
		t.Error("down with nothing to list closed the screen")
	}

	if _, out := Open().Key(press("esc"), e); !out.Leave {
		t.Error("esc with nothing to list did not close the screen")
	}
}

// TestTheListMarksWhatIsFilteredAndSaysWhenThereIsNothing.
func TestTheListMarksWhatIsFilteredAndSaysWhenThereIsNothing(t *testing.T) {
	e := world(t)
	e.Filter = List(e)[0].Name

	if drawn := strings.Join(Open().View(20, 100, e), "\n"); !strings.Contains(drawn, "[filtered]") {
		t.Errorf("the row the board is filtered to is not marked:\n%s", drawn)
	}

	e.Board, e.Filter = board.Board{}, ""

	if drawn := strings.Join(Open().View(20, 100, e), "\n"); !strings.Contains(drawn, "no repositories") {
		t.Errorf("an empty list says nothing about being empty:\n%s", drawn)
	}
}

// TestOneRowCarriesTheCursorAndEveryRowStartsInTheSameColumn. The mark is
// what says which row a keystroke will act on, so a second one is a reader
// acting on the wrong repository — and a mark that does not take the width
// of the gutter it stands in steps its own row a cell out of the column the
// rest of the list is in.
func TestOneRowCarriesTheCursorAndEveryRowStartsInTheSameColumn(t *testing.T) {
	e := world(t)

	s, _ := Open().Key(press("down"), e)

	var rows []string

	for _, l := range s.View(20, 100, e) {
		if stripped := ansi.Strip(l); strings.Contains(stripped, "to do ·") {
			rows = append(rows, stripped)
		}
	}

	if len(rows) != len(List(e)) {
		t.Fatalf("the list drew %d rows for %d repositories", len(rows), len(List(e)))
	}

	marked := 0

	for i, row := range rows {
		if strings.HasPrefix(row, cells.Mark) {
			marked++

			if i != s.sel {
				t.Errorf("row %d carries the cursor and row %d is the one chosen", i, s.sel)
			}
		}

		// Whatever stands in the gutter, the name after it starts in the
		// same column on every row.
		if at := len([]rune(row)) - len([]rune(strings.TrimLeft(row, " "+cells.Mark))); at != cells.Gutter {
			t.Errorf("row %d starts its name at cell %d, want %d", i, at, cells.Gutter)
		}
	}

	if marked != 1 {
		t.Errorf("%d rows carry the cursor, want one", marked)
	}
}

// TestAListWithNoRowsToDrawDrawsNothing, which is what every window passes
// through while somebody drags its corner.
func TestAListWithNoRowsToDrawDrawsNothing(t *testing.T) {
	e := world(t)

	for _, h := range []int{-1, 0} {
		if got := Open().View(h, 100, e); got != nil {
			t.Errorf("a list %d rows tall drew %d lines, want none", h, len(got))
		}
	}

	if got := Open().View(1, 100, e); len(got) == 0 {
		t.Error("a list with a row in it drew nothing")
	}
}
