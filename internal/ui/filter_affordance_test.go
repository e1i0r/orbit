package ui

// filter_conditions_coverage_test.go is trimLastRune's empty-string guard
// and affordance's "not offered at all" answer — the one branch a window
// asking about its own bindings never takes, since every verb the menu
// might ask about is on Keys.Affordances' list.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

func TestTrimLastRune(t *testing.T) {
	if got := trimLastRune(""); got != "" {
		t.Errorf("trimLastRune(\"\") = %q, want empty", got)
	}

	if got := trimLastRune("café"); got != "caf" {
		t.Errorf("trimLastRune(\"café\") = %q, want \"caf\" (a rune, not a byte, trimmed)", got)
	}
}

func TestAffordanceNotFound(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	// NeedsYou is expanded by default; a collapsed band has no task rows
	// to put the cursor on at all.
	m = at(t, m, view.NeedsYou, false)

	row, ok := m.selected()
	if !ok {
		t.Fatalf("expected a selected task row")
	}
	// Back names no verb in Keys.Affordances' list, so asking about it
	// answers false rather than matching the wrong entry.
	_, found := m.affordance(row.task, m.keys.Back)
	if found {
		t.Errorf("expected affordance(Back) to find nothing")
	}
}
