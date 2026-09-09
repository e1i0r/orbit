package arch

// A line of code stays inside the column, and the number that do not only
// goes down.
//
// The rule is 100 columns, and it is about code: a sentence a person reads —
// a translated string, a prompt, a URL — is exempt, because breaking one to
// fit a column makes it harder to read, not easier, and puts the catalogue's
// keys and the window's words out of step with each other.
//
// It is a ratchet rather than a wall because the rule arrived after the code
// did. The budget below is what was over the line the day it was written; a
// change that brings it down edits the number, and a change that pushes it
// up fails here until the lines it added are wrapped.

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// column is where a line of code has to end.
const column = 100

// overTheColumn is how many lines of code are still past it. It came down
// from here and it does not go back up.
const overTheColumn = 256

// quoted is a string literal, whatever is inside it. What a person reads is
// measured by them and not by this test.
var quoted = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)

// TestCodeStaysInsideTheColumn.
func TestCodeStaysInsideTheColumn(t *testing.T) {
	over := 0

	for _, path := range goFiles(t) {
		over += overLongIn(t, path)
	}

	if over > overTheColumn {
		t.Errorf("%d lines of code are past column %d, and the budget is %d — wrap what you added",
			over, column, overTheColumn)
	}

	if over < overTheColumn {
		t.Errorf("%d lines of code are past column %d and the budget still says %d — bring the number down to %d, the ratchet only turns one way",
			over, column, overTheColumn, over)
	}
}

// overLongIn counts the lines of one file that are code and past the column.
func overLongIn(t *testing.T, path string) int {
	t.Helper()

	over, raw := 0, false

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	for _, line := range strings.Split(string(body), "\n") {
		ticks := strings.Count(line, "`")

		if len(line) > column && !raw && len(quoted.ReplaceAllString(line, `""`)) > column {
			over++
		}

		if ticks%2 == 1 {
			raw = !raw
		}
	}

	return over
}
