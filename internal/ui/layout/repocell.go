package layout

// What the repository column of a row says, now that a task is worked in as
// many checkouts as it needed rather than in one.
//
// It is here and not in the window because the column is planned here: the
// width the plan hands out is measured off the same text the row draws, and
// two rules for one string is how a table comes to overrun the terminal it
// was planned for.

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// RepoCell is what the repository column says about one task in cells
// columns.
//
// A task is one row however many checkouts it reached into, so the cell is a
// count before it is a name: `payments +2` is a task worked in three
// repositories, and the two it does not name are the two there was no room
// for. Given the room it names all of them — which is why this takes a
// width. The column is sized by the short form, so the whole list appears in
// the rows that fit inside what some other row's short form already paid
// for.
func RepoCell(t view.Task, cells int) string {
	names := repoNames(t)
	if len(names) < 2 {
		return repoShort(t)
	}

	if whole := strings.Join(names, ", "); lipgloss.Width(whole) <= cells {
		return whole
	}

	return repoShort(t)
}

// repoShort is the cell at its narrowest: the repository the task is filed
// under, and how many more it is worked in.
func repoShort(t view.Task) string {
	names := repoNames(t)
	if len(names) == 0 {
		return ""
	}

	if len(names) == 1 {
		return names[0]
	}

	return names[0] + " +" + strconv.Itoa(len(names)-1)
}

// repoNames is every repository a task is worked in, and it reads the one
// name a row carries when it carries no list.
//
// A Task built by hand — a fixture, a golden, a row a test types out — names
// its repository in Repo and stops there, and a column that read only the
// list would draw nothing for it. The board fills both.
func repoNames(t view.Task) []string {
	if len(t.Repos) > 0 {
		return t.Repos
	}

	if t.Repo == "" {
		return nil
	}

	return []string{t.Repo}
}
