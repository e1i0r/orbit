package ui

// update_test.go is the transition table from the plan, walked row by row.
//
// The table is the specification of Update: each row names a key or a
// message, the precondition it arrives under, and what must be true after.
// A wrong row is visible as a wrong row; a wrong case in a ninety-line
// switch is invisible. Every Cmd asserted on is left unexecuted or has a
// port supplied by this test, so walking the table opens and starts nothing.

import (
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

func TestTheTransitionTable(t *testing.T) {
	cases := []struct {
		name  string
		start func(t *testing.T) Model
		msg   tea.Msg
		want  func(t *testing.T, m Model, cmd tea.Cmd)
	}{{
		name:  "tickMsg refreshes and ticks again",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return m },
		msg:   tickMsg(fixtureNow),
		want: func(t *testing.T, _ Model, cmd tea.Cmd) {
			batch, ok := cmd().(tea.BatchMsg)
			if !ok || len(batch) != 2 {
				t.Fatalf("tickMsg returned %T with %d commands, want a batch of the refresh and the next tick", cmd(), len(batch))
			}
		},
	}, {
		name:  "boardMsg with a crossing rings once",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return m },
		msg:   boardMsg{Board: fixtureBoard(fixtureTasks(), 4), Changed: board.Changed{Entered: []string{"ACME-2662"}}},
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			if !m.notified || cmd == nil {
				t.Errorf("notified=%v cmd=%v, want a bell rung once", m.notified, cmd != nil)
			}
		},
	}, {
		name: "the first boardMsg never rings",
		start: func(t *testing.T) Model {
			return New(Options{Words: words.For("en"), Settings: &settings{}, Width: 100, Height: 30})
		},
		msg: boardMsg{Board: fixtureBoard(fixtureTasks(), 4)},
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			if m.notified || cmd != nil {
				t.Errorf("notified=%v cmd=%v, want silence on the first refresh", m.notified, cmd != nil)
			}
		},
	}, {
		name:  "a boardMsg that loses the cursor's task clamps the cursor",
		start: lastRow,
		msg:   boardMsg{Board: fixtureBoard(fixtureTasks()[:3], 4)},
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.cursor >= len(m.rows()) {
				t.Errorf("cursor is %d of %d rows, want it clamped onto the list", m.cursor, len(m.rows()))
			}
			if _, ok := m.selected(); !ok {
				t.Error("the cursor is on nothing selectable after the list shrank")
			}
		},
	}, {
		name:  "elapsedMsg moves the clock and asks for the next one",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return m },
		msg:   elapsedMsg(fixtureNow.Add(time.Minute)),
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			if !m.now.Equal(fixtureNow.Add(time.Minute)) || cmd == nil {
				t.Errorf("now=%v cmd=%v, want the clock moved and asked again", m.now, cmd != nil)
			}
		},
	}, {
		name:  "down moves the cursor",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return m },
		msg:   press("down"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.cursor != 2 {
				t.Errorf("cursor is %d, want 2 — the second task row", m.cursor)
			}
		},
	}, {
		name:  "down on the last row does nothing, and does not wrap",
		start: lastRow,
		msg:   press("j"),
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			if m.cursor != len(m.rows())-1 || cmd != nil {
				t.Errorf("cursor is %d of %d and cmd=%v, want it held on the last row", m.cursor, len(m.rows()), cmd != nil)
			}
		},
	}, {
		name:  "enter on a row opens the task view",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return m },
		msg:   press("enter"),
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			if m.screen != screenDetail || cmd == nil {
				t.Errorf("screen=%v cmd=%v, want the task view and a diff command", m.screen, cmd != nil)
			}
		},
	}, {
		name:  "a diff for a task the reader has already left is dropped",
		start: func(t *testing.T) Model { return openOn(t, "ACME-2701") },
		msg:   diffMsg{ID: "ACME-2662", Text: "diff --git a/webhook.go b/webhook.go"},
		want:  func(t *testing.T, m Model, _ tea.Cmd) { wantNoPane(t, m) },
	}, {
		name:  "enter on a collapsed band opens it in place",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return at(t, m, view.Done, true) },
		msg:   press("enter"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if !m.expanded[view.Done] || m.screen != screenList {
				t.Errorf("expanded=%v screen=%v, want the band open and the list still on screen", m.expanded[view.Done], m.screen)
			}
		},
	}, {
		name:  "pause on a live task asks the command to pause it",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return at(t, m, view.Running, false) },
		msg:   press("p"),
		want: func(t *testing.T, _ Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Fatal("pause on a live task returned no command")
			}
			got, ok := cmd().(controlMsg)
			if !ok || got.ID != "ACME-2705" || got.Word != "pause" {
				t.Errorf("the command produced %#v, want a controlMsg pausing ACME-2705", cmd())
			}
		},
	}, {
		name:  "pause on a task that is not running is refused, verbatim",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return m },
		msg:   press("p"),
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			refusal := words.For("en").T("why.pause_not_running", "pausing needs a running task; nothing is running here")
			if cmd != nil || m.message != refusal {
				t.Errorf("cmd=%v message=%q, want no command and %q", cmd != nil, m.message, refusal)
			}
		},
	}, {
		name:  "cancel on a live task asks first",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return at(t, m, view.Running, false) },
		msg:   press("x"),
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			if m.confirm != confirmCancel || cmd != nil {
				t.Errorf("confirm=%v cmd=%v, want the confirm open and nothing cancelled yet", m.confirm, cmd != nil)
			}
		},
	}, {
		name:  "autopilot flips the setting and says which way",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return m },
		msg:   press("A"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			off := words.For("en").T("msg.autopilot_off", "autopilot is off: every phase stops for you")
			if m.opts.Settings.Autopilot() || m.message != off {
				t.Errorf("autopilot=%v message=%q, want it off and %q", m.opts.Settings.Autopilot(), m.message, off)
			}
		},
	}, {
		name:  "slash opens the filter",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return m },
		msg:   press("/"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if !m.filtering {
				t.Error("slash did not open the filter")
			}
		},
	}, {
		name:  "quit ends the program",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return m },
		msg:   press("q"),
		want: func(t *testing.T, _ Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Fatal("q returned no command")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Errorf("q produced %T, want tea.QuitMsg", cmd())
			}
		},
	}, {
		name: "quit with a confirm open closes the confirm instead",
		start: func(t *testing.T) Model {
			m, _ := testModel(t, 100, 30)
			m.confirm, m.confirmID = confirmCancel, "ACME-2705"
			return m
		},
		msg: press("q"),
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			if m.confirm != confirmNone || cmd != nil {
				t.Errorf("confirm=%v cmd=%v, want the confirm closed and the program still running", m.confirm, cmd != nil)
			}
		},
	}, {
		name:  "a terminal at least sixty columns wide reflows",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return m },
		msg:   tea.WindowSizeMsg{Width: 80, Height: 24},
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.tooNarrow || m.frame.Body.W != 80 || m.plan.Width() > 80 {
				t.Errorf("tooNarrow=%v body=%d plan=%d, want a frame and a row plan for 80 columns", m.tooNarrow, m.frame.Body.W, m.plan.Width())
			}
		},
	}, {
		name:  "a terminal under sixty columns is refused, with the number",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return m },
		msg:   tea.WindowSizeMsg{Width: 40, Height: 24},
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if !m.tooNarrow || m.narrow.Need != 60 || m.narrow.Got != 40 {
				t.Errorf("tooNarrow=%v need=%d got=%d, want the refusal carrying both numbers", m.tooNarrow, m.narrow.Need, m.narrow.Got)
			}
		},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			next, cmd := c.start(t).Update(c.msg)
			m, ok := next.(Model)
			if !ok {
				t.Fatalf("Update returned %T, want ui.Model", next)
			}
			c.want(t, m, cmd)
		})
	}
}

// TestEveryMessageSaysSomething is the plan's one-screen question: does
// every message this program has reach a case in Update? It is asked by
// handing Update one of each and requiring an effect — a sentence in the
// band, a diff in the pane, a language changed. The task view is open
// because a diff is only taken for the task it is open on; a message with
// no case reaches the default and changes nothing, which this table refuses.
func TestEveryMessageSaysSomething(t *testing.T) {
	sent := errors.New("the runner is not listening")
	cases := []struct {
		name string
		msg  tea.Msg
		want func(t *testing.T, m Model, cmd tea.Cmd)
	}{{
		name: "a control that was accepted",
		msg:  controlMsg{ID: "ACME-2705", Word: "pause"},
		want: func(t *testing.T, m Model, _ tea.Cmd) { wantBand(t, m, "ACME-2705") },
	}, {
		name: "a control that was refused",
		msg:  controlMsg{ID: "ACME-2705", Word: "pause", Err: sent},
		want: func(t *testing.T, m Model, _ tea.Cmd) { wantBand(t, m, sent.Error()) },
	}, {
		name: "a run that started",
		msg:  startedMsg{ID: "ACME-2710", Pid: 4114},
		want: func(t *testing.T, m Model, _ tea.Cmd) { wantBand(t, m, "ACME-2710") },
	}, {
		name: "a run that would not start",
		msg:  startedMsg{ID: "ACME-2710", Err: sent},
		want: func(t *testing.T, m Model, _ tea.Cmd) { wantBand(t, m, sent.Error()) },
	}, {
		name: "an editor that came back badly",
		msg:  editorMsg{Err: sent},
		want: func(t *testing.T, m Model, _ tea.Cmd) { wantBand(t, m, sent.Error()) },
	}, {
		name: "a diff that arrived",
		msg:  diffMsg{ID: "ACME-2662", Text: "diff --git a/webhook.go b/webhook.go"},
		want: func(t *testing.T, m Model, _ tea.Cmd) { wantPane(t, m, "diff --git") },
	}, {
		name: "a language the reader chose",
		msg:  languageMsg{Lang: "es"},
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.opts.Settings.Language() != "es" || m.keys.Quit.Help().Desc != "salir" {
				t.Errorf("language=%q quit=%q, want the setting written and the key map rebuilt", m.opts.Settings.Language(), m.keys.Quit.Help().Desc)
			}
		},
	}, {
		name: "an enumeration that fired",
		msg:  rescanMsg(fixtureNow),
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Error("a rescanMsg produced no enumeration and no next tick")
			}
		},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := openOn(t, "ACME-2662")
			next, cmd := m.Update(c.msg)
			after, ok := next.(Model)
			if !ok {
				t.Fatalf("Update returned %T, want ui.Model", next)
			}
			c.want(t, after, cmd)
		})
	}
}
