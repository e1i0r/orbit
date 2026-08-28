package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
	var signedBy []string
	m.opts.RecordSupervisor = func(by, channel, message string) error {
		recorded = append(recorded, message)
		signedBy = append(signedBy, by)
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
	// The window used to sign every message with one particular person's
	// name. The thread is one conversation and every other door writes
	// "operator": a name hardcoded here made the same person read as two
	// participants depending on which door they came through.
	if len(signedBy) != 1 || signedBy[0] != "operator" {
		t.Errorf("signed by %v, want [operator]", signedBy)
	}
}

func TestSupervisorScrolling(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openSupervisor()
	m.supervisor.lines = []view.SupervisorLine{
		{At: time.Now(), By: "elio", Channel: "tui", Text: "msg1"},
		{At: time.Now(), By: "supervisor", Channel: "autopilot", Text: "msg2"},
	}
	m.supervisor.offset = 0

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

func TestSupervisorReplyMsgHandling(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openSupervisor()

	// Reply msg
	m = next(t, m, supervisorReplyMsg{Text: "all tasks healthy"})
	if m.supervisorBusy {
		t.Error("expected supervisorBusy to be false after reply")
	}
	if !strings.Contains(m.message, "supervisor replied") {
		t.Errorf("expected message notification, got: %q", m.message)
	}
}

// TestSupervisorRenderingMarksAWithdrawnLine: the window shows the same
// thread the supervisor's prompt is built from, so a line that no longer
// steers the supervisor must not read on screen like one that does.
func TestSupervisorRenderingMarksAWithdrawnLine(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openSupervisor()
	m.supervisor.lines = []view.SupervisorLine{{
		At:        time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC),
		By:        "elio",
		Channel:   "tui",
		Text:      "the one I regret",
		Retracted: true,
	}}

	full := strings.Join(m.supervisorRows(25, 100), "\n")
	if !strings.Contains(full, "retract") {
		t.Errorf("a withdrawn line is drawn like any other:\n%s", full)
	}
	if !strings.Contains(full, "the one I regret") {
		t.Errorf("a withdrawn line was hidden instead of marked:\n%s", full)
	}
}

// TestSupervisorScreenIsSquare is what the redesign is for. The thread, the
// rule under the title and the input box each used to work out their own
// width, so the screen came out crooked by a couple of cells in a way no
// single line looked wrong on its own. Every row of every box is now the
// same width, at any terminal size.
func TestSupervisorScreenIsSquare(t *testing.T) {
	for _, size := range []struct{ w, h int }{{120, 34}, {80, 24}, {60, 16}, {200, 50}} {
		m, _ := testModel(t, size.w, size.h)
		m = m.openSupervisor()
		m.supervisor.lines = []view.SupervisorLine{
			{At: time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC), By: "claude", Channel: "mcp", Text: "a reply long enough to wrap more than once on a narrow terminal, with a `code span` in it"},
			{At: time.Date(2026, 8, 28, 9, 1, 0, 0, time.UTC), By: "operator", Channel: "tui", TaskID: "ORB-1", Text: "short one"},
		}

		rows := m.supervisorRows(size.h, size.w)
		if len(rows) != size.h {
			t.Errorf("%dx%d: %d rows, want %d", size.w, size.h, len(rows), size.h)
		}
		want := -1
		for i, r := range rows {
			got := lipgloss.Width(r)
			if strings.TrimSpace(r) == "" {
				continue
			}
			if want == -1 {
				want = got
			}
			if got != want {
				t.Errorf("%dx%d: row %d is %d cells, the rest are %d: %q", size.w, size.h, i, got, want, r)
			}
			if got > size.w {
				t.Errorf("%dx%d: row %d is %d cells, wider than the terminal", size.w, size.h, i, got)
			}
		}
	}
}

// TestSupervisorPickingTakesTheChosenLineBack is the retraction, from the
// window: ^R picks, the arrows choose, ↵ withdraws.
func TestSupervisorPickingTakesTheChosenLineBack(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	first := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	second := first.Add(time.Minute)
	var asked []time.Time
	m.opts.RetractSupervisor = func(at time.Time) error {
		asked = append(asked, at)
		return nil
	}
	m = m.openSupervisor()
	m.supervisor.lines = []view.SupervisorLine{
		{At: first, By: "operator", Channel: "tui", Text: "the one I regret"},
		{At: second, By: "claude", Channel: "mcp", Text: "an answer"},
	}

	// Typing is off while picking: r on its own is a letter, ^R is the mode.
	m = next(t, m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if !m.supervisor.picking {
		t.Fatal("^R did not open the picker")
	}
	if m.supervisor.pick != 1 {
		t.Errorf("picker opened on line %d, want the last one", m.supervisor.pick)
	}

	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.supervisor.pick != 0 {
		t.Errorf("up moved to %d, want 0", m.supervisor.pick)
	}
	// And it stops at the top rather than walking off it.
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.supervisor.pick != 0 {
		t.Errorf("up past the first line moved to %d", m.supervisor.pick)
	}

	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(asked) != 1 || !asked[0].Equal(first) {
		t.Errorf("retracted %v, want [%v]", asked, first)
	}
	if m.supervisor.picking {
		t.Error("the picker stayed open after taking a line back")
	}
	if m.supervisor.input != "" {
		t.Errorf("picking typed into the message box: %q", m.supervisor.input)
	}
}

func TestSupervisorPickingCancelsAndRefusesWhatItCannotDo(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	called := 0
	m.opts.RetractSupervisor = func(at time.Time) error { called++; return nil }
	m = m.openSupervisor()

	// Nothing said yet: there is no line to point at.
	m = next(t, m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if m.supervisor.picking {
		t.Error("^R opened the picker over an empty thread")
	}

	m.supervisor.lines = []view.SupervisorLine{
		{At: time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC), By: "operator", Channel: "tui", Text: "said once", Retracted: true},
	}
	m = next(t, m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.supervisor.picking {
		t.Error("esc left the picker open")
	}
	if called != 0 {
		t.Error("esc took a line back")
	}

	// A line already withdrawn is not withdrawn twice.
	m = next(t, m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if called != 0 {
		t.Errorf("a line already taken back was retracted again (%d calls)", called)
	}
}
