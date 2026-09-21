package view

// What a model wrote, folded into one row.
//
// A tool call's arguments are JSON a model produced: it may be indented, it
// may hold a path with backslashes in it, it may be a whole heredoc, and it
// may not be JSON at all. What comes out goes on one line of a table beside
// five other fields, so the one thing it can never be is more than one line.

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/record"
)

// FuzzAToolCallIsOneRow.
func FuzzAToolCallIsOneRow(f *testing.F) {
	for _, seed := range []string{
		"", `{"command":"go test ./..."}`, `{"command": "go test ./..."}`,
		`{"file_path":"C:\\Users\\ana\\main.go"}`, `{"command":"echo \"hi\""}`,
		"{\"command\":\"grep -rn \\\\\\n  needle .\"}", "not json at all",
		`{"pattern":"needle","-i":true}`, `{"style":"table"}`,
		"{\"command\":\"cd internal\\ngo test ./...\"}",
		`{"file_path":"/Users/ana/.orbit/worktrees/abc/ACME-1/a.go"}`,
		"{\"command\":\"a\\rb\"}", "{\"command\":\"a\\tb\"}",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, args string) {
		got := ToolLine("Bash", args)

		// The name is what a reader scans the column for, and a row that
		// lost it is a row about nothing.
		if !strings.HasPrefix(got, "Bash") {
			t.Errorf("ToolLine(%q) = %q, which does not open with the tool's name", args, got)
		}

		// One row. A line break in it silently adds a row to the table and
		// a carriage return redraws the one above it.
		if strings.ContainsAny(got, "\n\r") {
			t.Errorf("ToolLine(%q) = %q, which is more than one row", args, got)
		}

		if utf8.ValidString(args) && !utf8.ValidString(got) {
			t.Errorf("ToolLine(%q) = %q, which is not text", args, got)
		}
	})
}

// FuzzARunFoldsIntoARow.
//
// Fold is handed whatever the record holds — a run that was killed, an
// engine that printed nothing, a log an older Orbit wrote — and every
// surface there is draws what it answers. A reading that fell over on one
// task takes the whole board with it.
func FuzzARunFoldsIntoARow(f *testing.F) {
	f.Add("task.created", "", "", 1)
	f.Add("phase.started", "implement", "claude", 3)
	f.Add("task.failed", "review", "", 2)
	f.Add("", "", "", 0)
	f.Add("phase.tool_call", "implement", "Bash", 12)

	f.Fuzz(func(t *testing.T, kind, phase, text string, n int) {
		if n < 0 || n > 32 {
			t.Skip()
		}

		at := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)

		events := make([]record.Event, 0, n+1)
		events = append(events, record.Event{Kind: record.TaskCreated, At: at})

		for i := range n {
			events = append(events, record.Event{
				Kind: kind, At: at.Add(time.Duration(i) * time.Minute),
				Phase: phase, Text: text,
			})
		}

		row := Fold(events)

		// Whatever band it lands in is one the board actually draws. A row
		// in a band nothing knows about is a task that disappears: every
		// surface walks Bands() and shows what is in each.
		band := BandOf(row)

		drawn := false

		for _, one := range Bands() {
			if one == band {
				drawn = true
			}
		}

		if !drawn {
			t.Errorf("a run of %d %q events folds into %v, which no surface draws", n, kind, band)
		}

		// Nothing a row carries is more than one line: every one of these
		// is drawn in a column beside others.
		for what, value := range map[string]string{
			"title": row.Title, "action": row.CurrentAction, "thought": row.CurrentThought,
		} {
			if strings.ContainsAny(value, "\n\r") {
				t.Errorf("a run of %d %q events folds into a %s of more than one line: %q",
					n, kind, what, value)
			}
		}
	})
}
