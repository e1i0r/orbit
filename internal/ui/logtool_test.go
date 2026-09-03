package ui

// What a tool call reads as on the timeline: the argument it was made with,
// and not the document that argument was written in.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// calling is the record with one tool call in it, made with the arguments
// given.
func calling(tool, args string) []view.Entry {
	return append(fixtureEntries(), view.Entry{
		At: ago(20 * time.Minute), Kind: "phase.tool_call", Phase: "implement",
		Attempt: 2, PhaseN: 1, Tool: tool, Text: args,
	})
}

// TestAToolCallOnTheTimelineIsItsArgumentAndNotItsDocument.
//
// The timeline had a reader of its own that searched the arguments for
// `"command"` and then for `:"`. A call about a file matches neither, so
// every read, write and search on the tab arrived as raw JSON — the same
// prefix on every row, and the path buried in the middle of it.
func TestAToolCallOnTheTimelineIsItsArgumentAndNotItsDocument(t *testing.T) {
	for _, tc := range []struct {
		name string
		tool string
		args string
		want string
	}{
		{
			"a file read under the worktree the run was given",
			"Read",
			`{"file_path":"/Users/ana/.orbit/worktrees/5a14f7401345/ACME-2662/internal/task/run.go"}`,
			"Read: internal/task/run.go",
		},
		{
			"a search names what it is searching for",
			"Grep",
			`{"pattern":"needle","output_mode":"files_with_matches"}`,
			"Grep: needle",
		},
		{
			"a command written by an encoder, with the space after the colon",
			"Bash",
			`{"command": "go test ./...", "timeout": 120}`,
			"Bash: go test ./...",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, lines := timeline(t, calling(tc.tool, tc.args))

			if rowOf(lines, tc.want) < 0 {
				t.Errorf("no row reads %q:\n%s", tc.want, strings.Join(lines, "\n"))
			}

			for i, l := range lines {
				if strings.Contains(l, `"file_path"`) || strings.Contains(l, `"pattern"`) || strings.Contains(l, `"timeout"`) {
					t.Errorf("row %d shows the document around the argument: %q", i, l)
				}
			}
		})
	}
}
