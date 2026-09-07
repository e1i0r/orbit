package repos

// The repository list: what it collects, and every key it answers.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
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
