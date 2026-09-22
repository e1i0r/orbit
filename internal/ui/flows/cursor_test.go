package flows

// A cursor past the end of the list, which is what a flow saved or deleted
// in another window leaves behind.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// TestARowThatIsNotThereIsEditedAndDeletedByNothing. The list is read from
// the directory when the screen opens and again whenever it changes, so the
// cursor is an index into a list that can be shorter than it was — and
// every door that reads a row by index has an end on both sides.
func TestARowThatIsNotThereIsEditedAndDeletedByNothing(t *testing.T) {
	e := world(t)
	s := Open(FromBoard, e)

	n := len(s.listed)
	if n == 0 {
		t.Fatal("the fixture lists no flows")
	}

	for _, sel := range []int{-2, n, n + 5} {
		at := s
		at.sel = sel

		for _, key := range []tea.KeyPressMsg{
			{Code: 'e', Text: "e"},
			{Code: 'd', Text: "d"},
			{Code: tea.KeyEnter},
		} {
			next, out := at.Key(key, e)

			if next.Creating() || next.Previewing() {
				t.Errorf("%v on row %d of %d opened the designer", key, sel, n)
			}

			if next.confirmDelete {
				t.Errorf("%v on row %d of %d offered to delete something", key, sel, n)
			}

			if out.Leave {
				t.Errorf("%v on row %d of %d left the screen", key, sel, n)
			}
		}
	}

	// And the last row there is, is a row: the guard stops at what is not
	// there rather than at what is.
	last := s
	last.sel = n - 1

	if next, _ := last.Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e); !next.Previewing() {
		t.Error("⏎ on the last flow of the list did not open it")
	}
}

// TestTheCursorStartsOnTheCreateButton, which is what -1 means here: the
// rows that belong to no flow carry it too, so a page that pulled to it
// would jump to the floor.
func TestTheCursorStartsOnTheCreateButton(t *testing.T) {
	e := world(t)
	s := Open(FromBoard, e)

	if s.sel != -1 {
		t.Fatalf("the list opens on row %d, want the create button", s.sel)
	}

	drawn := ansi.Strip(strings.Join(s.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n"))
	if !strings.Contains(drawn, "Create Custom Flow") {
		t.Errorf("the create button is not on the screen:\n%s", drawn)
	}

	// And ⏎ on it opens the designer rather than a flow.
	next, _ := s.Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e)
	if !next.Creating() {
		t.Error("⏎ on the create button did not open the designer")
	}
}
