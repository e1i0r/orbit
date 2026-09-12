package known

// The tray: what you said that nobody has answered yet.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/words"
)

// kept is one sentence the window asked to have written down.
type kept struct {
	at            time.Time
	phrase, check string
}

// answers is what the two ports were asked for, so that a gesture is read
// back rather than guessed at.
type answers struct {
	keeps []kept
	drops []time.Time
}

// withTray is the screen open on a tray and whatever facts are already
// written down.
func withTray(t *testing.T, said []Said, facts ...knowledge.Fact) (State, Env, *answers) {
	t.Helper()

	got := &answers{}

	e := Env{
		Words:   words.For("en"),
		Keys:    keymap.New(words.For("en")),
		All:     func() []knowledge.Fact { return facts },
		Waiting: func() []Said { return said },
		Keep: func(at time.Time, phrase, check string) error {
			got.keeps = append(got.keeps, kept{at: at, phrase: phrase, check: check})

			return nil
		},
		Drop: func(at time.Time) error {
			got.drops = append(got.drops, at)

			return nil
		},
		Repo: "/w/orbit",
	}

	return Open(e), e, got
}

// saidAt is one sentence, at a moment of its own.
func saidAt(n int, text string) Said {
	return Said{At: time.Date(2026, 9, 11, 9, n, 0, 0, time.UTC), Text: text}
}

// TestTheTrayIsDrawnAboveWhatOrbitKnows.
//
// It is the one part of this screen with a question in it, and a question
// under two pages of facts is a question nobody answers.
func TestTheTrayIsDrawnAboveWhatOrbitKnows(t *testing.T) {
	s, e, _ := withTray(t,
		[]Said{saidAt(1, "never push without the tests passing")},
		known("errors are wrapped", knowledge.Scope{Kind: knowledge.General}))

	drawn := drawnKnowledge(t, s, e)

	tray := strings.Index(drawn, "never push without the tests passing")
	fact := strings.Index(drawn, "errors are wrapped")

	switch {
	case tray < 0:
		t.Fatalf("the tray is not on the screen:\n%s", drawn)
	case fact < 0:
		t.Fatalf("the facts are not on the screen:\n%s", drawn)
	case tray > fact:
		t.Error("the tray is drawn under what Orbit already knows")
	}
}

// TestKeepingOneIsOneKeystroke, because the answer to most of them is yes
// and a yes that takes an editor is a yes nobody gives.
func TestKeepingOneIsOneKeystroke(t *testing.T) {
	one := saidAt(1, "never push without the tests passing")

	s, e, asked := withTray(t, []Said{one})

	if _, out := s.Key(press("k"), e); out.Said == "" {
		t.Error("keeping a rule said nothing to the reader")
	}

	if len(asked.keeps) != 1 {
		t.Fatalf("%d rules were kept", len(asked.keeps))
	}

	if got := asked.keeps[0]; got.phrase != one.Text || !got.at.Equal(one.At) {
		t.Errorf("the window asked for %v to be kept", got)
	}
}

// TestCorrectingItIsHowMostOfTheseAreAccepted.
//
// What you meant is what you typed the second time, and being asked is worth
// nothing if the only answers are yes and no. The check is typed in the same
// breath: what a rule says and what makes it stop are one thought.
func TestCorrectingItIsHowMostOfTheseAreAccepted(t *testing.T) {
	s, e, asked := withTray(t, []Said{saidAt(1, "never push without tests")})

	s, _ = s.Key(press("e"), e)
	s = typed(s, e, " passing")
	s, _ = s.Key(press("tab"), e)
	s = typed(s, e, "make check")

	if _, out := s.Key(press("enter"), e); out.Said == "" {
		t.Error("keeping a corrected rule said nothing to the reader")
	}

	if len(asked.keeps) != 1 {
		t.Fatalf("%d rules were kept", len(asked.keeps))
	}

	got := asked.keeps[0]
	if got.phrase != "never push without tests passing" || got.check != "make check" {
		t.Errorf("the window asked for %q with check %q", got.phrase, got.check)
	}
}

// TestDroppingOneSaysItWasNotARule.
func TestDroppingOneSaysItWasNotARule(t *testing.T) {
	one := saidAt(1, "we never found out why that hung")

	s, e, asked := withTray(t, []Said{one})

	if _, out := s.Key(press("d"), e); out.Said == "" {
		t.Error("dropping a sentence said nothing to the reader")
	}

	if len(asked.drops) != 1 || !asked.drops[0].Equal(one.At) {
		t.Errorf("the window dropped %v", asked.drops)
	}

	if len(asked.keeps) != 0 {
		t.Error("a dropped sentence was written down as well")
	}
}

// TestTheCursorWalksTheTrayAndThenTheFacts.
//
// One cursor over two lists: past the last sentence it is on the first fact,
// and the keys that belong to a fact start answering. A cursor that stopped
// at the tray would leave the facts unreachable whenever anything was
// waiting.
func TestTheCursorWalksTheTrayAndThenTheFacts(t *testing.T) {
	s, e, asked := withTray(t,
		[]Said{saidAt(1, "never push without the tests passing")},
		known("errors are wrapped", knowledge.Scope{Kind: knowledge.General}))

	s, _ = s.Key(press("down"), e)

	if _, waiting := s.onSaid(); waiting {
		t.Fatal("one press down is still in the tray of one sentence")
	}

	f, ok := s.onFact()
	if !ok || f.Phrase != "errors are wrapped" {
		t.Fatalf("the cursor is on %v", f)
	}

	s, _ = s.Key(press("k"), e)

	if _, _ = s.Key(press("d"), e); len(asked.keeps)+len(asked.drops) != 0 {
		t.Error("the tray's keys answered for a fact the cursor was on")
	}
}

// TestTheWaysOutAreTheOnesUnderTheCursor. The two halves of this screen
// answer different questions, and a bar listing the keys of both is a bar
// nobody reads.
func TestTheWaysOutAreTheOnesUnderTheCursor(t *testing.T) {
	s, e, _ := withTray(t,
		[]Said{saidAt(1, "never push without the tests passing")},
		known("errors are wrapped", knowledge.Scope{Kind: knowledge.General}))

	if drawn := drawnKnowledge(t, s, e); !strings.Contains(drawn, "[k] keep it") {
		t.Errorf("the tray does not say how to keep one:\n%s", drawn)
	}

	s, _ = s.Key(press("down"), e)

	if drawn := drawnKnowledge(t, s, e); !strings.Contains(drawn, "[n] new") {
		t.Errorf("on a fact, the screen says:\n%s", drawn)
	}
}

// TestARowSaysWhereItWasSaid.
//
// The same words typed while correcting one run and said to the supervisor
// are the same rule, and which it was is how somebody decides whether it was
// meant that widely.
func TestARowSaysWhereItWasSaid(t *testing.T) {
	one := saidAt(1, "never merge without the tests passing")
	one.From = "PAY-1"

	s, e, _ := withTray(t, []Said{one})

	if drawn := drawnKnowledge(t, s, e); !strings.Contains(drawn, "PAY-1") {
		t.Errorf("the row does not say where it came from:\n%s", drawn)
	}
}

// TestAnEmptyTrayIsNotThere, because a heading over nothing is a question a
// reader has to answer for themselves.
func TestAnEmptyTrayIsNotThere(t *testing.T) {
	s, e, _ := withTray(t, nil,
		known("errors are wrapped", knowledge.Scope{Kind: knowledge.General}))

	if drawn := drawnKnowledge(t, s, e); strings.Contains(drawn, "You said this") {
		t.Errorf("an empty tray is drawn anyway:\n%s", drawn)
	}

	if s.Unanswered() != 0 {
		t.Error("the header is told there is something waiting")
	}
}
