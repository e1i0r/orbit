package ui

// Pasting with the gesture everybody actually uses.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// drawnLine is what the supervisor screen has on it, as plain text. The
// window can only reach the screen through its doors, so what a paste did is
// asserted where the operator would see it.
func drawnLine(t *testing.T, m Model) string {
	t.Helper()

	return ansi.Strip(strings.Join(m.supervisorRows(30, 100), "\n"))
}

// TestCmdVReachesTheSupervisorLine.
//
// ^V works and shells out to pbpaste, which is what the bar advertises. But
// nobody presses ^V: a terminal sends a real paste as bracketed paste, and
// with nothing listening for it the text was dropped where the operator could
// not tell whether they had pasted anything at all.
func TestCmdVReachesTheSupervisorLine(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.RecordSupervisor = func(string, string, string, string) error { return nil }
	m = m.openSupervisor()

	next := next(t, m, tea.PasteMsg{Content: "orbit/pull/104"})
	if drawn := drawnLine(t, next); !strings.Contains(drawn, "orbit/pull/104") {
		t.Errorf("what was pasted is not in the line:\n%s", drawn)
	}
}

// TestPastingManyLinesKeepsThemAll. A paste is usually the reason somebody
// wants more than one line, and the input already takes them.
func TestPastingManyLinesKeepsThemAll(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.RecordSupervisor = func(string, string, string, string) error { return nil }
	m = m.openSupervisor()

	next := next(t, m, tea.PasteMsg{Content: "first\nsecond\nthird"})

	drawn := drawnLine(t, next)
	for _, want := range []string{"first", "second", "third"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the line lost %q of a three line paste:\n%s", want, drawn)
		}
	}
}

// TestAPasteOutsideAFieldChangesNothing. On the board there is nothing to
// paste into, and text arriving as keystrokes there would be a filter nobody
// opened.
func TestAPasteOutsideAFieldChangesNothing(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	next := next(t, m, tea.PasteMsg{Content: "nothing to do with this"})
	if next.filter != "" {
		t.Errorf("a paste on the board left %q in the filter", next.filter)
	}

	if drawn := drawnLine(t, next.openSupervisor()); strings.Contains(drawn, "nothing to do with this") {
		t.Errorf("a paste on the board reached the supervisor's line:\n%s", drawn)
	}
}
