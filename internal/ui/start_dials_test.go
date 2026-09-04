package ui

// start_startdials_coverage_test.go is the start dialog's own key map and
// the two knobs beside it: cycleThinking's toggle, and newStart's three
// shapes — a task with no flow of its own, one whose flow the cycle already
// has, and one whose flow resolves to nothing at all.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

func TestNewStartEveryShape(t *testing.T) {
	// 1. No flow of its own: falls back to the default.
	s := newStart(nil, view.Task{ID: "X-1"})
	if s.chosen().name != "task" {
		t.Errorf("newStart with no flow chose %q, want the default", s.chosen().name)
	}

	// 2. A flow already in the built-in cycle: at points at it.
	s = newStart(nil, view.Task{ID: "X-1", Flow: "quick"})
	if s.chosen().name != "quick" || s.chosen().err != nil {
		t.Errorf("newStart(quick) = %+v, want it resolved with no error", s.chosen())
	}

	// 3. A flow nothing answers to: prepended, and carries its own error.
	s = newStart(nil, view.Task{ID: "X-1", Flow: "not-a-real-flow"})
	if s.chosen().name != "not-a-real-flow" || s.chosen().err == nil {
		t.Errorf("newStart(not-a-real-flow) = %+v, want it prepended with an error", s.chosen())
	}
}

func TestStartKeyEveryBinding(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onto(t, m, "ACME-2698") // a NeedsYou task, expanded by default and not live
	next, _ := m.openStart()

	m = asModel(t, next)
	if m.screen != screenStart {
		t.Fatalf("openStart left screen %v, want screenStart", m.screen)
	}

	presses := []string{"+", "m", "o", "t"}
	for _, k := range presses {
		next, _ := m.startKey(keystroke(k))
		m = asModel(t, next)
	}
	// openFlows/openEngines/cycleEffort/cycleThinking each moved the screen
	// or a knob; return to the start screen for the rest of the walk.
	m.screen = screenStart

	before := m.start.at
	next, _ = m.startKey(keystroke("f"))

	m = asModel(t, next)
	if len(m.start.flows) > 1 && m.start.at == before {
		t.Error("ChangeFlow ('f') did not move the cycle")
	}

	next, _ = m.startKey(keystroke("M"))
	m = asModel(t, next)
	m.screen = screenStart

	next, _ = m.startKey(keystroke("a"))
	m = asModel(t, next)

	next, _ = m.startKey(keystroke("?"))
	m = asModel(t, next)

	next, _ = m.tipKey(keystroke("?"))

	m = asModel(t, next)
	if m.screen != screenHelp {
		t.Errorf("Help ('? ?') left screen %v, want screenHelp", m.screen)
	}

	m.screen = screenStart

	next, cmd := m.startKey(tea.KeyPressMsg{Code: 'q', Text: "q"})

	m = asModel(t, next)
	if cmd == nil {
		t.Error("Quit did not produce a command")
	}

	next, _ = m.startKey(keystroke("esc"))

	m = asModel(t, next)
	if m.screen != screenList || m.start.id != "" {
		t.Errorf("Back left screen=%v start=%+v, want screenList and a cleared dialog", m.screen, m.start)
	}
}

func TestRunItRefusesAtTheCapAndOnAGoneTask(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. The task named on the dialog has since left the board.
	m.start = startModel{id: "no-such-task"}
	next, cmd := m.runIt()

	got := asModel(t, next)
	if cmd != nil {
		t.Error("runIt on a gone task produced a command")
	}

	wantBand(t, got, "has left the board")

	// 2. The unread cap refuses, and names the waiting tasks.
	m = onto(t, m, "ACME-2698")
	next, _ = m.openStart()
	m = asModel(t, next)
	m.opts.Settings = &settings{autopilot: true, lang: "en", unread: 1}
	next, cmd = m.runIt()

	got = asModel(t, next)
	if cmd != nil {
		t.Error("runIt at the unread cap produced a command")
	}

	if !strings.Contains(got.message, "press esc") {
		t.Errorf("runIt at the cap said %q, want the refusal naming the escape route", got.message)
	}
}

func TestCycleThinkingToggles(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	tests := []struct {
		start, want string
	}{
		{"", "off"},
		{"thinking", "off"},
		{"on", "off"},
		{"off", "thinking"},
	}
	for _, tt := range tests {
		m.knobs.Thinking = tt.start

		got := m.cycleThinking()
		if got.knobs.Thinking != tt.want {
			t.Errorf("cycleThinking from %q = %q, want %q", tt.start, got.knobs.Thinking, tt.want)
		}
	}
}
