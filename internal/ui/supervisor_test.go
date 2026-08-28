package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

func TestSupervisorScreenOpenAndAbandon(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// Open supervisor via S key
	m = step(t, m, "S")
	if m.screen != screenSupervisor {
		t.Fatalf("m.screen = %v, want screenSupervisor", m.screen)
	}

	// Esc returns to list screen
	m = step(t, m, "esc")
	if m.screen != screenList {
		t.Errorf("m.screen = %v, want screenList after esc", m.screen)
	}
}

func TestSupervisorTypingAndSubmit(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	var recorded []string
	m.opts.RecordSupervisor = func(by, channel, message string) error {
		recorded = append(recorded, message)
		return nil
	}

	m = m.openSupervisor()

	// Type message
	m = next(t, m, tea.KeyPressMsg{Text: "keep"})
	m = next(t, m, tea.KeyPressMsg{Text: " "})
	m = next(t, m, tea.KeyPressMsg{Text: "going"})
	if m.supervisor.input != "keep going" {
		t.Errorf("supervisor.input = %q, want 'keep going'", m.supervisor.input)
	}

	// Backspace
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.supervisor.input != "keep goin" {
		t.Errorf("after backspace input = %q", m.supervisor.input)
	}

	// Submit with Enter
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(recorded) != 1 || recorded[0] != "keep goin" {
		t.Errorf("recorded = %v, want ['keep goin']", recorded)
	}
	if m.supervisor.input != "" {
		t.Errorf("input not cleared after enter: %q", m.supervisor.input)
	}
}

func TestSupervisorScrolling(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openSupervisor()
	m.supervisor.lines = []view.SupervisorLine{
		{At: time.Now(), By: "elio", Channel: "tui", Text: "msg1"},
		{At: time.Now(), By: "supervisor", Channel: "autopilot", Text: "msg2"},
	}

	// Down increases offset
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.supervisor.offset != 1 {
		t.Errorf("offset after Down = %d, want 1", m.supervisor.offset)
	}

	// Up decreases offset
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.supervisor.offset != 0 {
		t.Errorf("offset after Up = %d, want 0", m.supervisor.offset)
	}
}

func TestSupervisorRendering(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openSupervisor()

	// Render empty
	rows := m.supervisorRows(25, 100)
	if len(rows) != 25 {
		t.Errorf("rows count = %d, want 25", len(rows))
	}
	full := strings.Join(rows, "\n")
	if !strings.Contains(full, "Supervisor") {
		t.Errorf("render did not contain Supervisor title: %s", full)
	}

	// Render with messages
	m.supervisor.lines = []view.SupervisorLine{
		{
			At:      time.Date(2026, 8, 28, 14, 0, 0, 0, time.UTC),
			By:      "supervisor",
			Channel: "autopilot",
			TaskID:  "ORB-1",
			Text:    "task ORB-1 completed all gates",
		},
	}
	rowsPopulated := m.supervisorRows(25, 100)
	fullPopulated := strings.Join(rowsPopulated, "\n")
	if !strings.Contains(fullPopulated, "ORB-1") || !strings.Contains(fullPopulated, "completed all gates") {
		t.Errorf("render with lines missing content:\n%s", fullPopulated)
	}
}
