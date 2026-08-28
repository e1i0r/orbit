package view

import (
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// SupervisorLine is one line of conversation, directive or action in the supervisor thread.
type SupervisorLine struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	By      string    `json:"by"`      // "elio", "operator", "supervisor", "claude", "opencode", "codex"
	Channel string    `json:"channel"` // "tui", "cli", "autopilot", "mcp"
	TaskID  string    `json:"task_id,omitempty"`
	Repo    string    `json:"repo,omitempty"`
	Text    string    `json:"text"`

	// Retracted is whether a later line took this one back. The line stays
	// in the thread — it was said, and a reader working out how the
	// supervisor got where it is needs to see it — but it is no longer
	// repeated into the model's prompt, and a reader should show it as
	// withdrawn rather than as standing advice.
	Retracted bool `json:"retracted,omitempty"`
}

// SupervisorThread folds raw supervisor events into viewable dialogue lines.
//
// A retraction is not dialogue: it is bookkeeping about a line above it, so
// it marks that line and does not become one of its own.
func SupervisorThread(events []record.Event) []SupervisorLine {
	gone := record.Retracted(events)
	lines := make([]SupervisorLine, 0, len(events))
	for _, e := range events {
		if e.Kind == record.SupervisorRetracted {
			continue
		}
		by := e.Data["by"]
		if by == "" {
			by = "operator"
		}
		channel := e.Data["channel"]
		if channel == "" {
			channel = "tui"
		}
		lines = append(lines, SupervisorLine{
			At:        e.At,
			Kind:      e.Kind,
			By:        by,
			Channel:   channel,
			TaskID:    e.Data["task_id"],
			Repo:      e.Data["repo"],
			Text:      e.Text,
			Retracted: gone[record.Stamp(e.At)],
		})
	}
	return lines
}
