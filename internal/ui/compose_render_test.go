package ui

// compose_render_test.go is the form as a frame: what a reader is asked for
// when they write a task down.
//
// It is a golden and not a list of assertions because what this screen is
// specified by is the number of things on it. The repository used to be the
// first of them, and the way that field is kept out is not a check anybody
// remembers to write — it is a frame that changes the day a field comes
// back.

import "testing"

// TestTheFormIsTheScreenItWasSpecifiedAs. Three fields in the manual tab —
// the flow, the id, the task — and two in the tab that reads an issue URL.
func TestTheFormIsTheScreenItWasSpecifiedAs(t *testing.T) {
	for _, c := range []struct {
		name string
		lang string
		tab  int
	}{
		{"compose-100x30-en", "en", composeTabManual},
		{"compose-100x30-es", "es", composeTabManual},
		{"compose-url-100x30-en", "en", composeTabURL},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := modelWith(t, printerFor(t, c.lang), fixtureBoard(fixtureTasks(), 4), 100, 30, nil)
			m = m.openCompose()
			m.compose.tab = c.tab

			golden(t, c.name, renderAt(t, m, 100, 30))
		})
	}
}
