package verb

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// TestAVerbNothingAnswersToIsRefused, because the name arrives from a
// command line, a URL and a tool call, and all three can carry a typo.
func TestAVerbNothingAnswersToIsRefused(t *testing.T) {
	_, err := Run(context.Background(), nil, "mergre", In{Task: "ACME-1"})
	if err == nil || !strings.Contains(err.Error(), "mergre") {
		t.Errorf("a name nothing answers to gave %v", err)
	}
}

// TestWhatAVerbNeedsIsAskedForOnce. "A note needs something written in it"
// is a fact about the verb; four copies of it were four chances for one of
// them to let an empty note through.
func TestWhatAVerbNeedsIsAskedForOnce(t *testing.T) {
	for _, one := range []struct {
		verb string
		in   In
		want string
	}{
		{"task note", In{Task: "ACME-1"}, "task note needs text"},
		{"task note", In{Args: map[string]string{"text": "hi"}}, "task note needs a task"},
		{"task direct", In{Task: "ACME-1"}, "task direct needs text"},
		{"board new", In{Args: map[string]string{"text": "do it"}}, "board new needs id"},
		{"supervisor say", In{}, "supervisor say needs text"},
	} {
		_, err := Run(context.Background(), nil, one.verb, one.in)
		if err == nil || !strings.Contains(err.Error(), one.want) {
			t.Errorf("%s with %v gave %v, want %q", one.verb, one.in.Args, err, one.want)
		}
	}
}

// TestWhatIsNeededIsRefusedBeforeAnythingIsWritten. The check runs before
// the world is touched, so a verb that was asked for wrongly leaves no
// half-written record behind.
func TestWhatIsNeededIsRefusedBeforeAnythingIsWritten(t *testing.T) {
	w := &counting{}

	if _, err := Run(context.Background(), w, "task note", In{Task: "ACME-1"}); err == nil {
		t.Fatal("an empty note was accepted")
	}

	if w.reached {
		t.Error("the world was reached for a verb that was refused")
	}
}

// counting is a world that says whether anything reached it.
type counting struct {
	World

	reached bool
}

func (c *counting) Looked() error { c.reached = true; return errors.New("no") }

// Words is answered rather than left to the embedded nil, because the
// refusal a verb makes before it reaches the world is a sentence a reader
// reads, and reading it is what this test is about.
func (c *counting) Words() *words.Printer { return words.For("") }

// TestAYesIsOnlyAYes. The field arrives as text from three surfaces, and
// "false" reaching a restart as true would start a run nobody asked for.
func TestAYesIsOnlyAYes(t *testing.T) {
	for word, want := range map[string]bool{
		"true": true, "yes": true, "1": true, "on": true,
		"false": false, "no": false, "": false, "maybe": false, "0": false,
	} {
		if got := (In{Args: map[string]string{"x": word}}).Yes("x"); got != want {
			t.Errorf("%q read as %v", word, got)
		}
	}
}
