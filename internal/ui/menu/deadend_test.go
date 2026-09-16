package menu

// A family on the menu that cannot be drilled into.

import (
	"strings"
	"testing"
)

// TestNoFamilyOnTheBoardsMenuIsADeadEnd.
//
// Five of them were: settings, rules, board, supervisor and knowledge all
// have children, so the top drew them as families — and the submenu was
// built from a table that only lists the two about a task, so choosing one
// landed on nothing at all.
func TestNoFamilyOnTheBoardsMenuIsADeadEnd(t *testing.T) {
	e := world(t)
	s := Open("", e)

	for _, row := range s.entries(e) {
		if row.Family == "" {
			continue
		}

		inside := s.moved(row.Family, e)
		if got := inside.entries(e); len(got) == 0 {
			t.Errorf("the %q family opens an empty menu", row.Family)
		}
	}
}

// TestAFamilyOffersItsOwnWordAndItsChildren. `settings` lists them and
// `settings set` changes one: a submenu with only the second would have lost
// the reading the parent is.
func TestAFamilyOffersItsOwnWordAndItsChildren(t *testing.T) {
	e := world(t)
	inside := Open("", e).moved("settings", e)

	var titles []string
	for _, row := range inside.entries(e) {
		titles = append(titles, row.Title)
	}

	joined := strings.Join(titles, " ")
	for _, want := range []string{"settings", "set", "clear"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the settings menu does not offer %q: %v", want, titles)
		}
	}
}

// TestAnEmptyMenuWithNoTaskDoesNotBlameATask. A menu opened on no task
// cannot have emptied because a task left the board, and saying so sent a
// reader looking for one they never had.
func TestAnEmptyMenuWithNoTaskDoesNotBlameATask(t *testing.T) {
	e := world(t)

	// A family nothing answers to, which is the only way a board menu is
	// empty now.
	empty := Open("", e).moved("nothing-at-all", e)

	drawn := strings.Join(empty.View(10, 80, e), " ")
	if strings.Contains(drawn, "no longer on the board") {
		t.Errorf("an empty board menu blamed a task: %q", strings.TrimSpace(drawn))
	}

	if !strings.Contains(drawn, "nothing under this one") {
		t.Errorf("an empty board menu says %q", strings.TrimSpace(drawn))
	}

	// And one opened on a task that has gone still says so.
	onGone := Open(gone, e).moved("task", e)
	if said := strings.Join(onGone.View(10, 80, e), " "); !strings.Contains(said, "no longer on the board") {
		t.Errorf("a menu on a task that left says %q", strings.TrimSpace(said))
	}
}
