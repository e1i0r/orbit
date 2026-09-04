package ui

// The menu inside a task carries both blocks, and the cursor never lands on
// a heading.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// inTaskMenu is the menu open on a task that is being looked at.
func inTaskMenu(t *testing.T) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{
		{ID: "ORBIT-5", Repo: "orbit", RepoPath: ".", Title: "a run to look at"},
	}
	m, _ = m.openDetail(m.board.Tasks[0])

	return asModel(t, must(m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})))
}

func must(model tea.Model, _ tea.Cmd) tea.Model { return model }

// TestTheMenuInsideATaskOffersItsVerbsToo. Reading a run and acting on it
// happen in the same place, and the verbs used to be a screen back.
func TestTheMenuInsideATaskOffersItsVerbsToo(t *testing.T) {
	m := inTaskMenu(t)

	var panes, verbs, heads int

	for _, e := range m.menuEntries() {
		switch {
		case e.head && e.title != "":
			heads++
		case e.tab != nil:
			panes++
		case e.aff != nil:
			verbs++
		}
	}

	if heads != 2 {
		t.Errorf("the menu has %d headings, want one over the panes and one over the verbs", heads)
	}

	if panes != int(tabCount) || verbs == 0 {
		t.Errorf("the menu has %d panes and %d verbs, want every pane and the task's verbs", panes, verbs)
	}
}

// TestTheCursorSkipsTheHeadings, both when the menu opens and when it is
// walked across the line between the two blocks — a cursor parked on a
// heading is a row that looks chosen and does nothing.
func TestTheCursorSkipsTheHeadings(t *testing.T) {
	m := inTaskMenu(t)

	es := m.menuEntries()
	if es[m.menu.sel].head {
		t.Fatalf("the menu opened on %+v, want the first entry there is to choose", es[m.menu.sel])
	}

	for i := 0; i < len(es); i++ {
		m = m.menuPick(1)
		if got := m.menuEntries()[m.menu.sel]; got.head {
			t.Fatalf("walking down landed on %+v after %d moves, want a row that does something", got, i+1)
		}
	}

	for i := 0; i < len(es); i++ {
		m = m.menuPick(-1)
		if got := m.menuEntries()[m.menu.sel]; got.head {
			t.Fatalf("walking back up landed on %+v after %d moves, want a row that does something", got, i+1)
		}
	}
}

// TestWalkingDownReachesTheVerbs: the block below the panes is reachable
// from the keyboard, not only by pressing the verb's own key.
func TestWalkingDownReachesTheVerbs(t *testing.T) {
	m := inTaskMenu(t)
	for range m.menuEntries() {
		m = m.menuPick(1)
	}

	if got := m.menuEntries()[m.menu.sel]; got.aff == nil {
		t.Errorf("the bottom of the menu is %+v, want the last of the task's verbs", got)
	}
}

// TestAHeadingIsNotSomethingToClickOn.
func TestAHeadingIsNotSomethingToClickOn(t *testing.T) {
	m := inTaskMenu(t)

	head := -1

	for i, e := range m.menuEntries() {
		if e.head && e.title != "" {
			head = i
			break
		}
	}

	if head < 0 {
		t.Fatal("the menu has no heading to click on")
	}

	if got := m.hitMenu(gutter, m.frame.Body.Y+menuTitleRows+head); got.Kind != TargetNone {
		t.Errorf("clicking the heading answered %+v, want nothing", got)
	}
}

// TestTheHeadingsAreDrawnWhereTheEntriesAreCounted. The menu is hit-tested
// by counting rows from the title, so a heading that drew two lines would
// put every click below it on the wrong verb.
func TestTheHeadingsAreDrawnWhereTheEntriesAreCounted(t *testing.T) {
	m := inTaskMenu(t)

	rows := m.menuRows(m.frame.Body.H, m.frame.Body.W)
	es := m.menuEntries()

	for i, e := range es {
		if e.head || e.title == "" {
			continue
		}

		row := i + menuTitleRows
		if row >= len(rows) {
			break
		}

		if !strings.Contains(rows[row], e.title) {
			t.Fatalf("entry %d (%q) is not on the row it is counted at: %q", i, e.title, rows[row])
		}
	}
}

// TestTheWheelLandsOnSomethingChoosable. It moves several rows at a time,
// and one of those jumps lands on the line between the two blocks.
func TestTheWheelLandsOnSomethingChoosable(t *testing.T) {
	m := inTaskMenu(t)

	for range len(m.menuEntries()) {
		m = m.menuPick(wheelRows)
		if got := m.menuEntries()[m.menu.sel]; got.head {
			t.Fatalf("the wheel landed on %+v, want a row that does something", got)
		}
	}

	for range len(m.menuEntries()) {
		m = m.menuPick(-wheelRows)
		if got := m.menuEntries()[m.menu.sel]; got.head {
			t.Fatalf("the wheel back up landed on %+v, want a row that does something", got)
		}
	}
}

// TestTheMenuFollowsItsCursorDownAndBackUp: a task's panes and verbs
// together are more rows than a small window has, and a verb the reader
// cannot see is a verb they do not have.
func TestTheMenuFollowsItsCursorDownAndBackUp(t *testing.T) {
	m := inTaskMenu(t)

	last := len(m.menuEntries()) - 1
	for range m.menuEntries() {
		m = m.menuPick(1)
	}

	view := menuView(m.frame.Body.H)
	if off := m.menuOffset(last+1, view); m.menu.sel < off || m.menu.sel >= off+view {
		t.Errorf("the cursor is at %d and the window shows %d..%d", m.menu.sel, off, off+view)
	}

	rows := m.menuRows(m.frame.Body.H, m.frame.Body.W)
	if want := m.menuEntries()[m.menu.sel].title; !strings.Contains(strings.Join(rows, "\n"), want) {
		t.Errorf("the chosen entry %q is not on the screen it is chosen on", want)
	}

	for range m.menuEntries() {
		m = m.menuPick(-1)
	}

	if m.menu.offset != 0 {
		t.Errorf("the list came back to the top at offset %d, want 0", m.menu.offset)
	}
}

// TestAnEntryIsWhereItWasDrawnAfterScrolling, or a click lands on the row
// the reader was not looking at.
func TestAnEntryIsWhereItWasDrawnAfterScrolling(t *testing.T) {
	m := inTaskMenu(t)
	for range m.menuEntries() {
		m = m.menuPick(1)
	}

	if m.menu.offset == 0 {
		t.Skip("the whole menu fits in this window, so there is nothing to scroll")
	}

	rows := m.menuRows(m.frame.Body.H, m.frame.Body.W)

	drawn := m.menu.sel - m.menu.offset + menuTitleRows
	if !strings.Contains(rows[drawn], m.menuEntries()[m.menu.sel].title) {
		t.Fatalf("row %d is %q, want the chosen entry", drawn, rows[drawn])
	}

	got := m.hitMenu(gutter, m.frame.Body.Y+drawn)
	if want := m.menuEntries()[m.menu.sel].glyph; got.Key != want {
		t.Errorf("clicking the chosen row answered %+v, want the entry drawn there (%q)", got, want)
	}
}
