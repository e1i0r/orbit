package ui

import (
	"strings"
	"testing"
)

// TestOOnTheBoardSaysWhereItWorks. The cheat sheet lists o among what a
// task can be done, and on the board it did nothing and said nothing.
func TestOOnTheBoardSaysWhereItWorks(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	next, cmd := m.listKey(keystroke("o"))
	if got := asModel(t, next).message; cmd != nil || !strings.Contains(got, "only the diff tab") {
		t.Errorf("o on the board said %q, want where it opens a file", got)
	}
}
