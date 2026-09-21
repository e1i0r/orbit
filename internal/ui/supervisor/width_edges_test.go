package supervisor

// This screen at every width and height it is drawn at.
//
// It is four blocks — a heading, the thread, the offers, and the line being
// typed — with a fifth down the side when there is room for it, and every
// one of them is laid out by subtracting: the gutter from the window, the
// mark from the gutter, the side column and its gap from what is left. A
// sum that is one out draws a row wider than the window it is in, which
// wraps, which puts every row under it where the reader is not looking.

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/view"
)

// TestEveryRowOfTheScreenFitsTheWindow, in every state the screen has and
// at every size a window can be dragged to.
func TestEveryRowOfTheScreenFitsTheWindow(t *testing.T) {
	kept := &held{lines: []view.SupervisorLine{
		spoke(fixtureNow.Add(-9*time.Minute), "c1", "operator", "use codex here"),
		spoke(fixtureNow.Add(-8*time.Minute), "c1", "zeta",
			"a reply long enough to be wrapped more than once in any window this test draws, "+
				"with a word nobody could break: palabralarguisimaquenoentraenningunacaja"),
		spoke(fixtureNow.Add(-7*time.Minute), "c2", "operator", "what about the ledger"),
	}}

	base := world(t, kept)
	base.Knows = func() []knowledge.Rule {
		return []knowledge.Rule{{
			ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
			Source: knowledge.Human, Phrase: "never log a card number, not even the last four",
		}}
	}

	for _, c := range []struct {
		name  string
		shape func(State, Env) State
	}{
		{"the thread", func(s State, _ Env) State { return s }},
		{"a line half typed", func(s State, _ Env) State {
			s.input = "una frase larga que sigue"

			return s
		}},
		{"the offers up", func(s State, _ Env) State { return s.Type("/") }},
		{"the conversations", func(s State, e Env) State {
			next, _ := s.Key(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}, e)

			return next
		}},
		{"picking a line", func(s State, e Env) State {
			next, _ := s.Key(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}, e)

			return next
		}},
	} {
		for w := layout.MinWidth; w <= 160; w += 3 {
			for _, h := range []int{14, 20, 30, 45} {
				frame, err := layout.Fit(w, h)
				if err != nil {
					continue
				}

				e := base
				e.Frame = frame

				s := c.shape(Open(0, e), e)

				body := e.Frame.Body
				rows := s.View(body.H, body.W, e)

				if len(rows) != body.H {
					t.Fatalf("%s in a body of %d rows drew %d", c.name, body.H, len(rows))
				}

				for i, r := range rows {
					if got := lipgloss.Width(r); got > body.W {
						t.Fatalf("%s at %dx%d: row %d is %d cells wide in a body of %d",
							c.name, w, h, i, got, body.W)
					}
				}
			}
		}
	}
}

// TestAScreenWithNoRoomDrawsNothing, which is what every window passes
// through while somebody drags its corner.
func TestAScreenWithNoRoomDrawsNothing(t *testing.T) {
	s, e := opened(t, saidThree())

	for _, h := range []int{-3, 0} {
		if got := s.View(h, e.Frame.Body.W, e); got != nil {
			t.Errorf("a screen %d rows tall drew %d rows", h, len(got))
		}
	}

	if got := s.View(1, e.Frame.Body.W, e); len(got) != 1 {
		t.Errorf("a screen one row tall drew %d rows", len(got))
	}
}
