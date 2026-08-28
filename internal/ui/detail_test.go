package ui

// detail_test.go is the task view's half of the transition table, walked row
// by row in the shape update_test.go established.

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

func TestTheTaskViewTransitionTable(t *testing.T) {
	cases := []struct {
		name  string
		start func(t *testing.T) Model
		msg   tea.Msg
		want  func(t *testing.T, m Model, cmd tea.Cmd)
	}{{
		name:  "⏎ on a task opens the task view on the overview",
		start: func(t *testing.T) Model { m, _ := testModel(t, 100, 30); return onto(t, m, "ACME-2662") },
		msg:   press("enter"),
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			if m.screen != screenDetail || m.detail != "ACME-2662" || m.tab != tabOverview {
				t.Errorf("screen=%v detail=%q tab=%v, want the task view open on overview", m.screen, m.detail, m.tab)
			}

			if cmd == nil {
				t.Fatal("opening the task view asked for neither the log nor the diff")
			}
			// Both reads are asked for, not merely something. A batch that
			// only ever ran the log would still pass a bare cmd != nil, and
			// the diff pane would open on a screen it never actually asked
			// about.
			batch, ok := cmd().(tea.BatchMsg)
			if !ok {
				t.Fatalf("opening the task view returned %T, want a batch of commands", cmd())
			}

			var sawLog, sawDiff bool

			for _, c := range batch {
				switch c().(type) {
				case logMsg:
					sawLog = true
				case diffMsg:
					sawDiff = true
				}
			}

			if !sawLog || !sawDiff {
				t.Errorf("opening the task view asked for log=%v diff=%v, want both", sawLog, sawDiff)
			}
		},
	}, {
		name:  "0 moves directly to the diff tab",
		start: func(t *testing.T) Model { m, _ := openDetail(t, "ACME-2662"); return m },
		msg:   press("0"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.tab != tabDiff {
				t.Errorf("tab is %v, want the diff", m.tab)
			}
		},
	}, {
		name: "tab wraps from the last tab back to the first",
		start: func(t *testing.T) Model {
			m, _ := openDetail(t, "ACME-2662")
			m.tab = tabCount - 1

			return m
		},
		msg: press("tab"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.tab != 0 {
				t.Errorf("tab is %v, want it wrapped round to the first tab", m.tab)
			}
		},
	}, {
		name: "shift+tab wraps backwards from the first tab to the last",
		start: func(t *testing.T) Model {
			m, _ := openDetail(t, "ACME-2662")
			m.tab = 0

			return m
		},
		msg: press("shift+tab"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.tab != tabCount-1 {
				t.Errorf("tab is %v, want it wrapped back to the last tab", m.tab)
			}
		},
	}, {
		name:  "esc returns to the list with the cursor where it was",
		start: func(t *testing.T) Model { m, _ := openDetail(t, "ACME-2698"); return m },
		msg:   press("esc"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.screen != screenList {
				t.Fatalf("screen is %v, want the list", m.screen)
			}

			r, ok := m.selected()
			if !ok || r.task.ID != "ACME-2698" {
				t.Errorf("the cursor came back on %+v, want the row it left from", r.task.ID)
			}
		},
	}, {
		name: "scrolling up lets the tail go",
		start: func(t *testing.T) Model {
			m, _ := openWith(t, "ACME-2662", longLog())
			return showing(t, m, tabTimeline)
		},
		msg: press("up"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.following {
				t.Error("the log is still following after the reader scrolled up")
			}
		},
	}, {
		name: "scrolling back to the last line takes the tail again",
		start: func(t *testing.T) Model {
			m, _ := openWith(t, "ACME-2662", longLog())
			m = showing(t, m, tabTimeline)

			return step(t, m, "up")
		},
		msg: press("down"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if !m.following {
				t.Error("the log is not following again after the reader scrolled back to the end")
			}
		},
	}, {
		name: "⏎ jumps the log to the newest entry and arms the tail",
		start: func(t *testing.T) Model {
			m, _ := openWith(t, "ACME-2662", longLog())
			m = showing(t, m, tabTimeline)

			return step(t, step(t, m, "up"), "up")
		},
		msg: press("enter"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if !m.following || !m.panes[tabLog].AtBottom() {
				t.Errorf("following=%v atBottom=%v, want the newest entry with the tail armed",
					m.following, m.panes[tabLog].AtBottom())
			}
		},
	}, {
		name: "end jumps to the newest entry from anywhere",
		start: func(t *testing.T) Model {
			m, _ := openWith(t, "ACME-2662", longLog())
			return step(t, step(t, m, "up"), "up")
		},
		msg: press("end"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if !m.following || !m.panes[tabLog].AtBottom() {
				t.Errorf("following=%v atBottom=%v, want end to jump to newest and re-arm",
					m.following, m.panes[tabLog].AtBottom())
			}
		},
	}, {
		name: "a tick while the task view is open re-reads the log",
		start: func(t *testing.T) Model {
			m, _ := openDetail(t, "ACME-2662")
			return m
		},
		msg: tickMsg(fixtureNow),
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			batch, ok := cmd().(tea.BatchMsg)
			if !ok || len(batch) != 3 {
				t.Fatalf("a tick under the task view returned %d commands, want the refresh, the next tick and the log", len(batch))
			}
		},
	}, {
		name: "a log that arrives for a task the reader has left is dropped",
		start: func(t *testing.T) Model {
			m, _ := openDetail(t, "ACME-2662")
			return m
		},
		msg: logMsg{ID: "ACME-2701", Entries: []view.Entry{{Kind: "task.created", Text: "another task"}}},
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if len(m.entries) != len(fixtureEntries()) {
				t.Errorf("the log pane took %d entries, want the %d it already had",
					len(m.entries), len(fixtureEntries()))
			}
		},
	}, {
		name: "a record that cannot be read is said in the pane, not swallowed",
		start: func(t *testing.T) Model {
			m, _ := openDetail(t, "ACME-2662")
			return m
		},
		msg: logMsg{ID: "ACME-2662", Err: errors.New("open events.jsonl: permission denied")},
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			wantIn(t, paneText(t, m), "permission denied")
		},
	}, {
		name:  "o outside the diff says why it did nothing",
		start: func(t *testing.T) Model { m, _ := openDetail(t, "ACME-2662"); return m },
		msg:   press("o"),
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			if cmd != nil {
				t.Error("o opened an editor from a tab that has no file in it")
			}

			wantBand(t, m, "diff")
		},
	}, {
		name: "o on the diff builds an editor and does not run it",
		start: func(t *testing.T) Model {
			t.Setenv("EDITOR", "vi")
			m, _ := openDetail(t, "ACME-2662")

			return step(t, m, "0")
		},
		msg: press("o"),
		want: func(t *testing.T, _ Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Fatal("o on the diff built no editor command")
			}
		},
	}, {
		name: "→ scrolls the diff sideways rather than leaving the screen",
		start: func(t *testing.T) Model {
			m, _ := openIn(t, words.For("en"), "ACME-2662", fixtureEntries(), wideDiff())
			return step(t, m, "0")
		},
		msg: press("right"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.screen != screenDetail {
				t.Fatalf("screen is %v, want the task view still open", m.screen)
			}

			if m.panes[tabDiff].XOffset() == 0 {
				t.Error("the diff did not scroll sideways")
			}
		},
	}, {
		name: "← comes back from a sideways scroll before it leaves the screen",
		start: func(t *testing.T) Model {
			m, _ := openIn(t, words.For("en"), "ACME-2662", fixtureEntries(), wideDiff())
			scrolled := step(t, step(t, m, "0"), "right")
			// XOffset() == 0 is also what a ← that had silently gone dead
			// would leave behind, so the row's own assertion cannot tell a
			// working ← from a broken one unless the scroll it is meant to
			// undo is known to have happened first.
			if scrolled.panes[tabDiff].XOffset() == 0 {
				t.Fatal("setup did not actually scroll the diff sideways before pressing ←")
			}

			return scrolled
		},
		msg: press("left"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.screen != screenDetail || m.panes[tabDiff].XOffset() != 0 {
				t.Errorf("screen=%v x=%d, want the diff scrolled back and the view still open",
					m.screen, m.panes[tabDiff].XOffset())
			}
		},
	}, {
		name:  "esc is Back on the log tab",
		start: func(t *testing.T) Model { m, _ := openDetail(t, "ACME-2662"); return m },
		msg:   press("esc"),
		want: func(t *testing.T, m Model, _ tea.Cmd) {
			if m.screen != screenList {
				t.Errorf("screen is %v after esc on the log tab, want the list", m.screen)
			}
		},
	}, {
		name: "a rescan while the task view is open re-reads the diff",
		start: func(t *testing.T) Model {
			m, _ := openDetail(t, "ACME-2662")
			return m
		},
		msg: rescanMsg(fixtureNow),
		want: func(t *testing.T, m Model, cmd tea.Cmd) {
			batch, ok := cmd().(tea.BatchMsg)
			if !ok || len(batch) != 3 {
				t.Fatalf("a rescan under the task view returned %d commands, want the rescan, the next rescan tick and the diff", len(batch))
			}
			// The count is not the assertion. The tickMsg case directly
			// above the line under test appends logOf under an identical
			// condition, so the likeliest wrong build here is a copy-paste
			// that asks for the log a second time — and it counts three
			// just as well. diffAsking is written by the branch that asks
			// for a diff and by nothing else, so it is what tells the two
			// apart. Running the commands would not: one of the three is
			// the next rescan tick, and calling it sits for two seconds.
			if !m.diffAsking {
				t.Error("a rescan under the task view left no diff outstanding, so the third command was not a diff")
			}
		},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := c.start(t)
			after, cmd := m.Update(c.msg)

			got, ok := after.(Model)
			if !ok {
				t.Fatalf("Update returned %T, want ui.Model", after)
			}

			c.want(t, got, cmd)
		})
	}
}
