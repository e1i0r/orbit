package settings

import (
	"strings"
	"testing"
)

// TestTheScreenDrawsItsTable.
func TestTheScreenDrawsItsTable(t *testing.T) {
	e := env(t, newFile())
	rows := Open(e).View(40, 100, e)

	if len(rows) != 40 {
		t.Errorf("the screen drew %d rows in 40", len(rows))
	}

	whole := ""
	for _, r := range rows {
		whole += r + "\n"
	}

	for _, want := range []string{"Settings", "language", "unread-cap", "edit"} {
		if !contains(whole, want) {
			t.Errorf("the screen never says %q", want)
		}
	}

	if none := Open(e).View(0, 100, e); none != nil {
		t.Errorf("a screen with no room drew %d rows", len(none))
	}
}

// contains is strings.Contains, named for what the tests are asking.
func contains(whole, part string) bool { return strings.Contains(whole, part) }

// TestTheLineIsDrawnWhereThePillsWere. While a row is being typed into, what
// it offers is not what matters — what is in the line is.
func TestTheLineIsDrawnWhereThePillsWere(t *testing.T) {
	e := env(t, newFile())
	rows := Open(e).Point(0).Edit("por-escribir").View(40, 100, e)

	whole := strings.Join(rows, "\n")
	if !strings.Contains(whole, "por-escribir") {
		t.Error("the screen does not show what is being typed")
	}

	if !strings.Contains(whole, "save") {
		t.Error("the way out does not say how to save while a line is open")
	}
}
