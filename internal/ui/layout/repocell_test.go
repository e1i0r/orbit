package layout

// What the repository column says about a task that was worked in more than
// one repository, and what it costs the rest of the row to say it.

import (
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// many is one task of three repositories beside one of a single repository,
// which is the board this file is about.
func many() []view.Task {
	return []view.Task{
		{ID: "ACME-2705", Repo: "app", Repos: []string{"app", "payments", "api"}, Title: "Reconciliation endpoint"},
		{ID: "ACME-2706", Repo: "payments", Repos: []string{"payments"}, Title: "Index on settlements"},
	}
}

// TestACellSaysHowManyRepositoriesBeforeItSaysWhich. The column is one field
// of a row and a task reaches into as many checkouts as the work needed, so
// the count is what always fits and the names are what fits when there is
// room.
func TestACellSaysHowManyRepositoriesBeforeItSaysWhich(t *testing.T) {
	three := many()[0]

	if got := RepoCell(three, 8); got != "app +2" {
		t.Errorf("a task of three repositories in eight cells says %q, want app +2", got)
	}

	if got := RepoCell(three, 40); got != "app, payments, api" {
		t.Errorf("a task of three repositories with room to spare says %q, want all three named", got)
	}

	if got := RepoCell(many()[1], 40); got != "payments" {
		t.Errorf("a task of one repository says %q, want the name on its own", got)
	}
}

// TestARowThatCarriesOneNameAndNoListStillDrawsIt. A Task built by hand — a
// fixture, a golden, a row a test types out — names its repository in Repo
// and stops there. A column that read only the list would draw an empty cell
// for every one of them.
func TestARowThatCarriesOneNameAndNoListStillDrawsIt(t *testing.T) {
	if got := RepoCell(view.Task{ID: "ACME-1", Repo: "payments"}, 20); got != "payments" {
		t.Errorf("a row with no list says %q, want payments", got)
	}

	if got := RepoCell(view.Task{ID: "ACME-1"}, 20); got != "" {
		t.Errorf("a row that names no repository at all says %q, want nothing", got)
	}
}

// TestTheColumnIsSizedByTheCountAndNotByTheList. Sizing it by the names would
// take those cells from the title on every board with a task in three
// repositories, which is the opposite of naming them only when there is room.
func TestTheColumnIsSizedByTheCountAndNotByTheList(t *testing.T) {
	p := Columns(200, many(), budget)
	if want := lipgloss.Width("payments"); p.Repo != want {
		t.Errorf("the repository column is %d cells, want the %d of the widest short form", p.Repo, want)
	}
}

// TestABoardWhereEveryRowSaysTheSameThingHasNoRepositoryColumn is the rule
// that was already there, read against the new cell: two tasks worked in the
// same two repositories say `app +1` twice, and a column that says one thing
// is cells taken from the title.
func TestABoardWhereEveryRowSaysTheSameThingHasNoRepositoryColumn(t *testing.T) {
	same := []view.Task{
		{ID: "ACME-2705", Repo: "app", Repos: []string{"app", "payments"}, Title: "Reconciliation endpoint"},
		{ID: "ACME-2706", Repo: "app", Repos: []string{"app", "payments"}, Title: "Index on settlements"},
	}

	if p := Columns(200, same, budget); p.Repo != 0 {
		t.Errorf("a board whose rows all say the same gave the repo column %d cells, want none", p.Repo)
	}
}
