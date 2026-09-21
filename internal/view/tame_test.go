package view

// What an engine prints and what a file says, on their way to a terminal.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// controls is every shape that is an instruction to a terminal rather than
// a character, and what each one does when it reaches one.
var controls = map[string]string{
	"\x1b[2J":           "clears the screen",
	"\x1b[31m":          "paints the rest of the row red",
	"\x1b]0;stolen\x07": "renames the window",
	"\x1b[1;1H":         "moves the cursor to the corner",
	"\a":                "rings the bell",
	"\r":                "takes the cursor back to the start of the row",
	"\x00":              "is a null byte",
	"\x7f":              "is a delete",
}

// TestWhatAnEnginePrintsCannotDriveTheTerminal. An entry's text is whatever
// the phase left on stdout, and the window hands it to a terminal — which
// reads an escape sequence as an instruction. A phase printing ESC[2J
// cleared the reader's screen on every frame that drew it.
func TestWhatAnEnginePrintsCannotDriveTheTerminal(t *testing.T) {
	for mark, does := range controls {
		events := []record.Event{
			{Kind: record.TaskCreated, Text: "a title that " + mark + "does"},
			{Kind: record.TaskStarted},
			{
				Kind: record.PhaseFinished, Phase: "implement", Text: "building " + mark + "and on",
				Data: map[string]string{"error": "it broke " + mark, "tool": "bash" + mark},
			},
		}

		for i, e := range Log(events) {
			for _, field := range []string{e.Text, e.Cause, e.Tool, e.Phase} {
				if strings.Contains(field, mark) {
					t.Errorf("entry %d carries %q, which %s: %q", i, mark, does, field)
				}
			}
		}

		if got := Fold(events).Title; strings.Contains(got, mark) {
			t.Errorf("the title carries %q, which %s: %q", mark, does, got)
		}
	}
}

// TestTamingLeavesTheWordsAlone. Everything a reader came for survives:
// the words, the line breaks the panes split on, and the accents and the
// emoji a tracker's title is full of.
func TestTamingLeavesTheWordsAlone(t *testing.T) {
	for _, c := range []struct{ from, want string }{
		{"plain words", "plain words"},
		{"two\nlines", "two\nlines"},
		{"decisión 決済 🔥", "decisión 決済 🔥"},
		{"a\ttab", "a tab"},
		{"\x1b[32mgreen\x1b[0m words", "green words"},
		{"\x1b]0;title\x07after", "after"},
		{"cut off mid-\x1b[", "cut off mid-"},
	} {
		if got := tame(c.from); got != c.want {
			t.Errorf("tame(%q) = %q, want %q", c.from, got, c.want)
		}
	}
}
