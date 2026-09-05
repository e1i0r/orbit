package ui

// What the supervisor's line offers while it is being typed.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// typing is a supervisor screen with something half written in it.
func typing(t *testing.T, input string) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m.opts.RecordSupervisor = func(string, string, string) error { return nil }
	m = m.openSupervisor()
	m.supervisor.input = input

	return m
}

// TestASlashOffersTheGesturesAndSaysWhatTheyDo.
//
// The gestures are worth nothing if nobody knows they exist. A slash is
// where somebody finds out, and each one has to say what it does — "/rule"
// and "/aware" are two words for two powers and the difference is the whole
// point.
func TestASlashOffersTheGesturesAndSaysWhatTheyDo(t *testing.T) {
	got := typing(t, "/").completions()

	if len(got) < 2 {
		t.Fatalf("a slash offered %d things: %+v", len(got), got)
	}

	seen := map[string]string{}
	for _, c := range got {
		seen[c.Text] = c.What
	}

	for _, want := range []string{ruleWord, awareWord} {
		if what, offered := seen[want]; !offered {
			t.Errorf("a slash does not offer %q", want)
		} else if what == "" {
			t.Errorf("%q is offered with nothing said about it", want)
		}
	}
}

// TestWhatIsTypedNarrowsWhatIsOffered.
func TestWhatIsTypedNarrowsWhatIsOffered(t *testing.T) {
	got := typing(t, "/aw").completions()
	if len(got) == 0 {
		t.Fatal("/aw offered nothing")
	}

	// Its own flags are still offered — that is where somebody learns they
	// exist — but nothing that is not the word being typed.
	for _, c := range got {
		if !strings.HasPrefix(c.Text, awareWord) {
			t.Errorf("/aw offered %q", c.Text)
		}
	}

	if none := typing(t, "/zzz").completions(); len(none) != 0 {
		t.Errorf("a gesture nobody has offered %+v", none)
	}
}

// TestAnAtOffersTheTasksOnTheBoard, by their id and with their title, since
// an id alone is not something anybody remembers.
func TestAnAtOffersTheTasksOnTheBoard(t *testing.T) {
	m := typing(t, "@")

	got := m.completions()
	if len(got) == 0 {
		t.Fatal("an at offered no tasks at all")
	}

	first := m.board.Tasks[0]
	for _, c := range got {
		if c.Text == atWord+first.ID {
			if c.What == "" {
				t.Errorf("%q is offered with no title beside it", c.Text)
			}

			return
		}
	}

	t.Errorf("the tasks on the board were not offered: %+v", got)
}

// TestOrdinaryTextOffersNothing. A conversation is not a command line, and a
// list popping up over somebody's sentence is the window interrupting them.
func TestOrdinaryTextOffersNothing(t *testing.T) {
	for _, said := range []string{"what happened", "", "/rule coverage stays above 90%", "look at ORB-1"} {
		if got := typing(t, said).completions(); len(got) != 0 {
			t.Errorf("%q offered %+v, want nothing", said, got)
		}
	}
}

// TestTabTakesWhatIsOffered, and leaves a space, so that the next thing
// typed is the sentence and not stuck to the gesture.
func TestTabTakesWhatIsOffered(t *testing.T) {
	m := typing(t, "/aw")

	next := next(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if next.supervisor.input != awareWord+" " {
		t.Errorf("tab left %q in the line, want %q", next.supervisor.input, awareWord+" ")
	}
}

// TestEnterTakesTheOfferRatherThanSending. The list is up because somebody
// is mid-word; sending half a gesture is never what they meant.
func TestEnterTakesTheOfferRatherThanSending(t *testing.T) {
	m := typing(t, "/ru")

	sent := false
	m.opts.RecordSupervisor = func(string, string, string) error {
		sent = true

		return nil
	}

	next := next(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if sent {
		t.Error("enter sent half a gesture instead of finishing it")
	}

	if !strings.HasPrefix(next.supervisor.input, ruleWord) {
		t.Errorf("enter left %q in the line", next.supervisor.input)
	}
}
