package known

// The line a fact is corrected in: what every key does to it, and what is
// drawn while it is open.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// correcting is the screen with one fact open for editing.
func correcting(t *testing.T, phrase string) (State, Env) {
	t.Helper()

	s, e := onScreen(t, knowledge.Fact{
		Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: phrase,
	})
	e.Replace = func(knowledge.Fact, knowledge.Fact) error { return nil }

	s, _ = s.Key(press("e"), e)
	if !s.editing {
		t.Fatal("e did not open the fact for editing")
	}

	return s, e
}

// TestEveryKeyOfTheLineDoesItsOwnThing. It is a field somebody types a
// sentence into, so the keys that move a caret have to be there: a line that
// only appends is a line where a typo means retyping the sentence.
func TestEveryKeyOfTheLineDoesItsOwnThing(t *testing.T) {
	s, e := correcting(t, "the fuzz tests hang")

	// The caret starts at the end, which is where a correction usually is.
	s, _ = s.Key(press("backspace"), e)
	if got := s.in[factPhrase].Val; got != "the fuzz tests han" {
		t.Errorf("backspace left %q", got)
	}

	s = typed(s, e, "g!")
	if got := s.in[factPhrase].Val; got != "the fuzz tests hang!" {
		t.Errorf("typing left %q", got)
	}

	// Home and the arrows put the caret where the correction is, and delete
	// takes the character in front of it.
	s, _ = s.Key(press("home"), e)
	s, _ = s.Key(press("delete"), e)

	if got := s.in[factPhrase].Val; got != "he fuzz tests hang!" {
		t.Errorf("delete at the head of the line left %q", got)
	}

	s, _ = s.Key(press("right"), e)
	s = typed(s, e, "-")

	if got := s.in[factPhrase].Val; got != "h-e fuzz tests hang!" {
		t.Errorf("typing one cell in from the head left %q", got)
	}

	s, _ = s.Key(press("left"), e)
	s, _ = s.Key(press("end"), e)
	s = typed(s, e, "?")

	if got := s.in[factPhrase].Val; got != "h-e fuzz tests hang!?" {
		t.Errorf("end then typing left %q", got)
	}
}

// TestTabIsTheOtherField, and comes back round: what a rule says and what
// makes it stop are one thought, and the two are typed one after the other.
func TestTabIsTheOtherField(t *testing.T) {
	s, e := correcting(t, "coverage stays above 90%")

	s, _ = s.Key(press("tab"), e)
	if s.field != factCheck {
		t.Fatalf("tab left the cursor on field %d, want the check", s.field)
	}

	s = typed(s, e, "make coverage")
	if got := s.in[factCheck].Val; got != "make coverage" {
		t.Errorf("what was typed after tab reads %q", got)
	}

	if s, _ = s.Key(press("tab"), e); s.field != factPhrase {
		t.Errorf("tab from the check left field %d, want back to the sentence", s.field)
	}
}

// TestAKeyTheLineHasNoUseForChangesNothing.
func TestAKeyTheLineHasNoUseForChangesNothing(t *testing.T) {
	s, e := correcting(t, "of everything")

	after, out := s.Key(press("up"), e)
	if after.in[factPhrase].Val != s.in[factPhrase].Val || out.Said != "" {
		t.Errorf("a key the line has no use for left %q and said %q",
			after.in[factPhrase].Val, out.Said)
	}
}

// TestASentenceEmptiedIsRefusedRatherThanWritten. A fact with nothing in it
// says nothing, and deleting one is not something this gesture does.
func TestASentenceEmptiedIsRefusedRatherThanWritten(t *testing.T) {
	s, e := correcting(t, "hi")

	wrote := false
	e.Replace = func(knowledge.Fact, knowledge.Fact) error {
		wrote = true

		return nil
	}

	s, _ = s.Key(press("backspace"), e)
	s, _ = s.Key(press("backspace"), e)

	after, out := s.Key(press("enter"), e)

	if wrote {
		t.Error("a fact with nothing left in it was written down")
	}

	if !after.editing {
		t.Error("the line closed on a sentence it refused to save")
	}

	if !strings.Contains(out.Said, "says nothing") {
		t.Errorf("the refusal reads %q", out.Said)
	}
}

// TestTheLineBeingTypedIntoCarriesTheCaret, and the other one does not: two
// blocks on screen is two places the next character could land.
func TestTheLineBeingTypedIntoCarriesTheCaret(t *testing.T) {
	s, e := correcting(t, "of everything")

	drawn := strings.Join(s.foot(80, e), "\n")
	if n := strings.Count(ansi.Strip(drawn), "█"); n != 1 {
		t.Errorf("the two fields carry %d carets between them:\n%s", n, ansi.Strip(drawn))
	}

	// The caret is drawn on the character it is on, not only after the last
	// one: a correction is made in the middle of a sentence.
	s, _ = s.Key(press("home"), e)

	drawn = ansi.Strip(strings.Join(s.foot(80, e), "\n"))
	if !strings.Contains(drawn, "of everything") {
		t.Errorf("the sentence is not on the line with the caret at its head:\n%s", drawn)
	}

	if n := strings.Count(drawn, "█"); n != 0 {
		t.Errorf("the caret at the head of the line drew %d blocks, want it on the character", n)
	}
}

// TestTheWaysOutSayWhichKeysTheLineAnswers, because a field with no verbs
// under it is a field somebody is stuck in.
func TestTheWaysOutSayWhichKeysTheLineAnswers(t *testing.T) {
	s, e := correcting(t, "of everything")

	drawn := ansi.Strip(strings.Join(s.foot(80, e), "\n"))
	for _, want := range []string{"tab", "save", "esc"} {
		if !strings.Contains(strings.ToLower(drawn), want) {
			t.Errorf("the ways out do not mention %q:\n%s", want, drawn)
		}
	}
}
