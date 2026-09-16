package chat

// One channel, served: who may ask, what is answered, and what is held back.

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// aChannel is a chat in memory: what it was told to hear, and what it said.
type aChannel struct {
	hears []Message

	mu   sync.Mutex
	said []string
}

func (c *aChannel) Name() string { return "test" }

func (c *aChannel) Listen(_ context.Context, said func(Message)) error {
	for _, m := range c.hears {
		said(m)
	}

	return nil
}

func (c *aChannel) Say(_ context.Context, _, text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.said = append(c.said, text)

	return nil
}

// served runs the desk over those messages and answers what it said back.
func served(t *testing.T, allowed func(string) bool, texts ...string) []string {
	t.Helper()

	c := &aChannel{}
	for _, one := range texts {
		c.hears = append(c.hears, Message{Where: "here", Who: "elio", Text: one})
	}

	// No world: every verb that reaches it fails, and every test here is
	// about what happens before or instead of that.
	e := Env{
		Words:   words.For("en"),
		Allowed: allowed,
		World: func() (verb.World, func(), error) {
			return nil, func() {}, errNoWorld
		},
	}

	if err := Open(e, c).Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	return c.said
}

var errNoWorld = errNoWorldType{}

type errNoWorldType struct{}

func (errNoWorldType) Error() string { return "no machine in this test" }

func anyone(string) bool { return true }

// TestOnlyTheOneAccountIsHeard. A chat anybody can join is a chat anybody can
// cancel a run from.
func TestOnlyTheOneAccountIsHeard(t *testing.T) {
	said := served(t, func(who string) bool { return who == "nobody" }, "/board")
	if len(said) != 0 {
		t.Errorf("a stranger was answered: %v", said)
	}

	// And a build that forgot to say who is allowed turns everybody away
	// rather than nobody.
	if said := served(t, nil, "/board"); len(said) != 0 {
		t.Errorf("with no gate at all it answered: %v", said)
	}
}

// TestHelpIsBuiltFromTheDeclaration. A hand-written command list is the list
// that goes stale the first time somebody adds a verb.
func TestHelpIsBuiltFromTheDeclaration(t *testing.T) {
	said := served(t, anyone, "/help")
	if len(said) != 1 {
		t.Fatalf("it answered %d times, want one", len(said))
	}

	for _, want := range []string{"/board new <id>", "/task start <id>", "/rules keep"} {
		if !strings.Contains(said[0], want) {
			t.Errorf("the help does not offer %q:\n%s", want, said[0])
		}
	}

	// And it does not offer what it cannot do.
	if strings.Contains(said[0], "/task take") {
		t.Errorf("the help offers a terminal a chat has not got:\n%s", said[0])
	}
}

// TestProseIsNotAnswered. Answering every stray sentence with a usage message
// is how a channel gets muted.
func TestProseIsNotAnswered(t *testing.T) {
	if said := served(t, anyone, "dale con codex"); len(said) != 0 {
		t.Errorf("prose was answered: %v", said)
	}
}

// TestWhatCannotBeTakenBackWaitsForASecondMessage.
func TestWhatCannotBeTakenBackWaitsForASecondMessage(t *testing.T) {
	said := served(t, anyone, "/pr merge ACME-3")
	if len(said) != 1 || !strings.Contains(said[0], "/yes") {
		t.Fatalf("a merge was not held for a confirmation: %v", said)
	}

	// The second message is the one that acts — and here it reaches the
	// world, which this test does not have. Failing there is the proof it
	// got past the hold.
	two := served(t, anyone, "/pr merge ACME-3", "/yes")
	if len(two) != 2 || !strings.Contains(two[1], "no machine") {
		t.Errorf("/yes did not run what was waiting: %v", two)
	}

	// And changing the subject clears it, so a confirmation cannot fire
	// later against something nobody was talking about any more.
	away := served(t, anyone, "/pr merge ACME-3", "/board", "/yes")
	if len(away) != 3 || strings.Contains(away[2], "no machine") {
		t.Errorf("a confirmation survived a change of subject: %v", away)
	}
}

// TestAVerbAChatCannotDoSaysWhyNot, rather than failing somewhere deeper.
func TestAVerbAChatCannotDoSaysWhyNot(t *testing.T) {
	said := served(t, anyone, "/task take ACME-3")
	if len(said) != 1 || !strings.Contains(said[0], "terminal") {
		t.Errorf("it answered %v", said)
	}
}
