package ui

// A line break inside an entry of the record. The timeline gives an entry one
// row and folds the rest of it away, and a break the drawing did not cut at
// is the rest of it arriving anyway — under the row on a newline, on top of
// it on a carriage return.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// brokenLines is the record with one tool call whose arguments carry each
// kind of break, which is how a command written on more than one line
// reaches the pane.
func brokenLines(ending string) []view.Entry {
	return append(fixtureEntries(), view.Entry{
		At: ago(20 * time.Minute), Kind: "phase.tool_call", Phase: "implement",
		Attempt: 2, PhaseN: 1, Tool: "Bash",
		Text: "make check" + ending + "go test ./internal/ui/",
	})
}

// TestAnEntryIsOneRowWhicheverWayItsLinesEnd. The row after the break is the
// half nobody asked for: a newline puts it under the entry, out of the
// columns it was drawn in, and a carriage return puts it over the entry, on
// top of the time and the phase beside it.
func TestAnEntryIsOneRowWhicheverWayItsLinesEnd(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ending string
	}{
		{"a newline", "\n"},
		{"a carriage return", "\r"},
		{"both, as an editor writes them", "\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, lines := timeline(t, brokenLines(tc.ending))

			y := rowOf(lines, "make check")
			if y < 0 {
				t.Fatalf("the tool call was not drawn:\n%s", strings.Join(lines, "\n"))
			}

			if strings.ContainsAny(lines[y], "\n\r") {
				t.Errorf("the row carries the break itself: %q", lines[y])
			}

			for i, l := range lines {
				if strings.Contains(l, "go test ./internal/ui/") {
					t.Errorf("row %d holds what came after the break: %q", i, l)
				}
			}
		})
	}
}
