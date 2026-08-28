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
}

// SupervisorThread folds raw supervisor events into viewable dialogue lines.
func SupervisorThread(events []record.Event) []SupervisorLine {
	lines := make([]SupervisorLine, 0, len(events))
	for _, e := range events {
		by := e.Data["by"]
		if by == "" {
			by = "operator"
		}
		channel := e.Data["channel"]
		if channel == "" {
			channel = "tui"
		}
		lines = append(lines, SupervisorLine{
			At:      e.At,
			Kind:    e.Kind,
			By:      by,
			Channel: channel,
			TaskID:  e.Data["task_id"],
			Repo:    e.Data["repo"],
			Text:    e.Text,
		})
	}
	return lines
}
