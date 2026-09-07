package supervisor

// What the supervisor's line offers while it is being typed.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/spoken"
)

// halfWritten is the screen with something half written in it, on a board
// of two tasks so that a mention has something to offer.
func halfWritten(t *testing.T, input string) (State, Env) {
	t.Helper()

	s, e := open(t)
	e.Tasks = []Mention{
		{ID: "ORB-1", Title: "the webhook retries"},
		{ID: "ORB-2", Title: "the report reads long"},
	}
	s.input = input

	return s, e
}

// TestASlashOffersTheGesturesAndSaysWhatTheyDo.
//
// The gestures are worth nothing if nobody knows they exist. A slash is
// where somebody finds out, and each one has to say what it does — "/rule"
// and "/aware" are two words for two powers and the difference is the whole
// point.
func TestASlashOffersTheGesturesAndSaysWhatTheyDo(t *testing.T) {
	s, e := halfWritten(t, "/")

	got := s.completions(e)

	if len(got) < 2 {
		t.Fatalf("a slash offered %d things: %+v", len(got), got)
	}

	seen := map[string]string{}
	for _, c := range got {
		seen[c.Text] = c.What
	}

	for _, want := range []string{spoken.RuleWord, spoken.AwareWord} {
		if what, offered := seen[want]; !offered {
			t.Errorf("a slash does not offer %q", want)
		} else if what == "" {
			t.Errorf("%q is offered with nothing said about it", want)
		}
	}
}

// TestWhatIsTypedNarrowsWhatIsOffered.
func TestWhatIsTypedNarrowsWhatIsOffered(t *testing.T) {
	s, e := halfWritten(t, "/aw")

	got := s.completions(e)
	if len(got) == 0 {
		t.Fatal("/aw offered nothing")
	}

	// Its own flags are still offered — that is where somebody learns they
	// exist — but nothing that is not the word being typed.
	for _, c := range got {
		if !strings.HasPrefix(c.Text, spoken.AwareWord) {
			t.Errorf("/aw offered %q", c.Text)
		}
	}

	if none, _ := halfWritten(t, "/zzz"); len(none.completions(e)) != 0 {
		t.Errorf("a gesture nobody has offered %+v", none.completions(e))
	}
}

// TestAnAtOffersTheTasksOnTheBoard, by their id and with their title, since
// an id alone is not something anybody remembers.
func TestAnAtOffersTheTasksOnTheBoard(t *testing.T) {
	s, e := halfWritten(t, "@")

	got := s.completions(e)
	if len(got) == 0 {
		t.Fatal("an at offered no tasks at all")
	}

	first := e.Tasks[0]
	for _, c := range got {
		if c.Text == spoken.AtWord+first.ID {
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
		s, e := halfWritten(t, said)
		if got := s.completions(e); len(got) != 0 {
			t.Errorf("%q offered %+v, want nothing", said, got)
		}
	}
}

// TestTabTakesWhatIsOffered, and leaves a space, so that the next thing
// typed is the sentence and not stuck to the gesture.
func TestTabTakesWhatIsOffered(t *testing.T) {
	s, e := halfWritten(t, "/aw")

	after, _ := s.Key(press("tab"), e)
	if after.input != spoken.AwareWord+" " {
		t.Errorf("tab left %q in the line, want %q", after.input, spoken.AwareWord+" ")
	}
}

// TestEnterTakesTheOfferRatherThanSending. The list is up because somebody
// is mid-word; sending half a gesture is never what they meant.
func TestEnterTakesTheOfferRatherThanSending(t *testing.T) {
	s, e := halfWritten(t, "/ru")

	kept := &held{}
	e.Record = kept.record

	after, _ := s.Key(press("enter"), e)

	if len(kept.wrote) != 0 {
		t.Error("enter sent half a gesture instead of finishing it")
	}

	if !strings.HasPrefix(after.input, spoken.RuleWord) {
		t.Errorf("enter left %q in the line", after.input)
	}
}
