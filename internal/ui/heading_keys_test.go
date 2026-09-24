package ui

import (
	"strings"
	"testing"
)

// TestTaskKeysOnAHeadingSayWhy. On a band's heading the keys about a task
// did nothing and said nothing, with [n] still drawn in the bar.
func TestTaskKeysOnAHeadingSayWhy(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.moveTo(0)

	if r, ok := m.selected(); !ok || !r.head {
		t.Fatalf("the first row is not a heading: %+v", r)
	}

	for _, k := range []string{"n", "p", "r", "s", "x", "b", "t", "h", "d", "D"} {
		next, _ := m.listKey(keystroke(k))
		if got := asModel(t, next).message; !strings.Contains(got, "heading") {
			t.Errorf("%s on a heading said %q, want why it did nothing", k, got)
		}
	}
}
