package ui

// start_test.go is the dialog that decides what a run will be before it
// runs: its transition table, the frame it is specified as, and the two
// refusals that are the whole point of asking first.
//
// Nothing here starts a run. The Start port is a closure that records what
// it was asked for and answers with a pid, which is what makes "press enter
// and the flow on screen is the flow that was passed" a fact this file can
// check without a process.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

// dialog is the model with the start dialog open on one task, reached the
// way a reader reaches it: put the cursor on the row and press the key.
//
// Naming startModel below is deliberate — this is the line that says the
// dialog has a state of its own, and it is the first thing to fail while
// there is no such type.
func dialog(t *testing.T, m Model, id string) (Model, startModel) {
	t.Helper()
	m = onRow(t, m, id)
	next, _ := m.Update(press("n"))
	opened, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", next)
	}
	if opened.screen != screenStart {
		t.Fatalf("screen is %v after n on %s, want the start dialog", opened.screen, id)
	}
	return opened, opened.start
}

// onRow puts the cursor on one task's row, by id.
func onRow(t *testing.T, m Model, id string) Model {
	t.Helper()
	for i, r := range m.rows() {
		if !r.head && !r.blank && r.task.ID == id {
			m.cursor = i
			return m
		}
	}
	t.Fatalf("no row for %s in the body", id)
	return m
}

// step hands one message to the window and returns the next model and its
// command, which is the shape every row of the tables below shares.
func advance(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	after, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", next)
	}
	return after, cmd
}

func TestTheStartDialogTransitionTable(t *testing.T) {
	cases := []struct {
		name  string
		start func(t *testing.T) (Model, *recorder)
		msg   tea.Msg
		want  func(t *testing.T, m Model, cmd tea.Cmd, got *recorder)
	}{{
		name: "n on a task opens the dialog on the flow the task was written for",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			return onRow(t, m, "ACME-2662"), got
		},
		msg: press("n"),
		want: func(t *testing.T, m Model, _ tea.Cmd, _ *recorder) {
			if m.screen != screenStart || m.start.id != "ACME-2662" {
				t.Fatalf("screen=%v id=%q, want the dialog open on ACME-2662", m.screen, m.start.id)
			}
			if name := m.start.chosen().name; name != "careful" {
				t.Errorf("the dialog opened on %q, want careful — the flow the task was written for", name)
			}
		},
	}, {
		name: "n on a running task is refused and the list stays on screen",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			return onRow(t, m, "ACME-2705"), got
		},
		msg: press("n"),
		want: func(t *testing.T, m Model, cmd tea.Cmd, _ *recorder) {
			if m.screen != screenList || cmd != nil {
				t.Errorf("screen=%v cmd=%v, want the board and nothing started", m.screen, cmd != nil)
			}
			wantBand(t, m, "ACME-2705")
		},
	}, {
		name: "n on a band header opens nothing and says nothing",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			return at(t, m, view.Done, true), got
		},
		msg: press("n"),
		want: func(t *testing.T, m Model, cmd tea.Cmd, _ *recorder) {
			if m.screen != screenList || cmd != nil {
				t.Errorf("screen=%v cmd=%v, want the board left alone", m.screen, cmd != nil)
			}
		},
	}, {
		name: "n with nothing to start says how a task is written",
		start: func(t *testing.T) (Model, *recorder) {
			return modelWith(t, printerFor(t, "en"), fixtureBoard(nil, 4), 100, 30, nil), nil
		},
		msg: press("n"),
		want: func(t *testing.T, m Model, cmd tea.Cmd, _ *recorder) {
			if m.screen != screenList || cmd != nil {
				t.Errorf("screen=%v cmd=%v, want the empty board left alone", m.screen, cmd != nil)
			}
			wantBand(t, m, "orbit new")
		},
	}, {
		name: "f moves to the next flow and redraws the phases under it",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			m, _ = dialog(t, m, "ACME-2662")
			return m, got
		},
		msg: press("f"),
		want: func(t *testing.T, m Model, _ tea.Cmd, _ *recorder) {
			f := m.start.chosen()
			if f.name != "quick" {
				t.Fatalf("f moved to %q, want quick — the next flow after careful", f.name)
			}
			if len(f.flow.Phases) != 1 {
				t.Errorf("quick draws %d phases, want 1 — the phase list follows the flow line", len(f.flow.Phases))
			}
		},
	}, {
		name: "f all the way round comes back to the task's own flow",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			m, _ = dialog(t, m, "ACME-2662")
			for range len(m.start.flows) - 1 {
				m, _ = advance(t, m, press("f"))
			}
			return m, got
		},
		msg: press("f"),
		want: func(t *testing.T, m Model, _ tea.Cmd, _ *recorder) {
			if name := m.start.chosen().name; name != "careful" {
				t.Errorf("the cycle ended on %q, want careful", name)
			}
		},
	}, {
		name: "esc closes the dialog and starts nothing",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			m, _ = dialog(t, m, "ACME-2662")
			return m, got
		},
		msg: press("esc"),
		want: func(t *testing.T, m Model, cmd tea.Cmd, got *recorder) {
			if m.screen != screenList || cmd != nil {
				t.Errorf("screen=%v cmd=%v, want the board back and nothing started", m.screen, cmd != nil)
			}
			if got.flow != "" {
				t.Errorf("the start port was asked for %q, want nothing", got.flow)
			}
		},
	}, {
		name: "A flips the standing switch from inside the dialog",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			m, _ = dialog(t, m, "ACME-2662")
			return m, got
		},
		msg: press("A"),
		want: func(t *testing.T, m Model, _ tea.Cmd, _ *recorder) {
			if m.screen != screenStart {
				t.Fatalf("screen=%v, want to still be in the dialog", m.screen)
			}
			if m.autopilotOn() {
				t.Error("autopilot is still on, want the switch flipped off")
			}
		},
	}, {
		name: "enter starts the flow on screen, with the unread count of the board it is showing",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			m, _ = dialog(t, m, "ACME-2662")
			m, _ = advance(t, m, press("f"))
			return m, got
		},
		msg: press("enter"),
		want: func(t *testing.T, m Model, cmd tea.Cmd, got *recorder) {
			if m.screen != screenList || cmd == nil {
				t.Fatalf("screen=%v cmd=%v, want the board back and a run asked for", m.screen, cmd != nil)
			}
			cmd()
			if got.id != "ACME-2662" || got.flow != "quick" {
				t.Errorf("started %q on %q, want ACME-2662 on quick — the flow that was on screen", got.id, got.flow)
			}
			if want := board.Unread(m.board); got.unread != want {
				t.Errorf("the cap was told %d unread, want %d — the board the window is showing", got.unread, want)
			}
		},
	}, {
		name: "enter at the unread cap is refused and names the tasks that are waiting",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := cappedModel(t)
			m, _ = dialog(t, m, "ACME-2662")
			return m, got
		},
		msg: press("enter"),
		want: func(t *testing.T, m Model, cmd tea.Cmd, got *recorder) {
			if cmd != nil || got.flow != "" {
				t.Fatalf("cmd=%v flow=%q, want nothing started at the cap", cmd != nil, got.flow)
			}
			for _, want := range []string{"ACME-2690", "ACME-2691", "ACME-2692"} {
				wantBand(t, m, want)
			}
		},
	}, {
		name: "enter without a way to start a run says so rather than doing nothing",
		start: func(t *testing.T) (Model, *recorder) {
			m, got := testModel(t, 100, 30)
			m.opts.Start = nil
			m, _ = dialog(t, m, "ACME-2662")
			return m, got
		},
		msg: press("enter"),
		want: func(t *testing.T, m Model, cmd tea.Cmd, _ *recorder) {
			if cmd == nil {
				t.Fatal("enter answered with no command at all, and a key that does nothing reads as broken")
			}
			after, _ := advance(t, m, cmd())
			if strings.TrimSpace(after.message) == "" {
				t.Error("the band says nothing about a window with no way to start a run")
			}
		},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, got := c.start(t)
			after, cmd := advance(t, m, c.msg)
			c.want(t, after, cmd, got)
		})
	}
}

// cappedModel is the fixture board with the unread cap already reached: the
// three finished tasks nobody has read are exactly the three the refusal has
// to name.
func cappedModel(t *testing.T) (Model, *recorder) {
	t.Helper()
	m, got := testModel(t, 100, 30)
	m.opts.Settings = &settings{autopilot: true, lang: "en", unread: 3}
	return m, got
}
