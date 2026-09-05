package ui

// Pasting with the gesture everybody actually uses.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestCmdVReachesTheSupervisorLine.
//
// ^V works and shells out to pbpaste, which is what the bar advertises. But
// nobody presses ^V: a terminal sends a real paste as bracketed paste, and
// with nothing listening for it the text was dropped where the operator could
// not tell whether they had pasted anything at all.
func TestCmdVReachesTheSupervisorLine(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.RecordSupervisor = func(string, string, string) error { return nil }
	m = m.openSupervisor()
	m.supervisor.input = "look at "

	next := next(t, m, tea.PasteMsg{Content: "https://github.com/e1i0r/orbit/pull/104"})
	if want := "look at https://github.com/e1i0r/orbit/pull/104"; next.supervisor.input != want {
		t.Errorf("the line holds %q, want %q", next.supervisor.input, want)
	}
}

// TestPastingManyLinesKeepsThemAll. A paste is usually the reason somebody
// wants more than one line, and the input already takes them.
func TestPastingManyLinesKeepsThemAll(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.RecordSupervisor = func(string, string, string) error { return nil }
	m = m.openSupervisor()

	next := next(t, m, tea.PasteMsg{Content: "first\nsecond\nthird"})
	if n := strings.Count(next.supervisor.input, "\n"); n != 2 {
		t.Errorf("the line holds %d newlines, want 2: %q", n, next.supervisor.input)
	}
}

// TestAPasteOutsideAFieldChangesNothing. On the board there is nothing to
// paste into, and text arriving as keystrokes there would be a filter nobody
// opened.
func TestAPasteOutsideAFieldChangesNothing(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	next := next(t, m, tea.PasteMsg{Content: "nothing to do with this"})
	if next.supervisor.input != "" || next.filter != "" {
		t.Errorf("a paste on the board left %q in the line and %q in the filter",
			next.supervisor.input, next.filter)
	}
}
