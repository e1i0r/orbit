package chat

// What an answer looks like once it is dressed for a chat.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// TestASentenceStaysASentence. A one-line answer put inside a monospaced
// block is a person being shouted at in a typewriter font.
func TestASentenceStaysASentence(t *testing.T) {
	got := Reply(verb.Out{Said: "noted on ACME-3"}, en())
	if got != "noted on ACME-3" {
		t.Errorf("it dressed a sentence as %q", got)
	}
}

// TestTerminalPaddingIsTakenOut is the whole of why a listing needs dressing
// at all: the verbs pad a column out to line up under the one above it,
// which is invisible on a wide screen and is a horizontal scroll bar over
// three words on a narrow one.
func TestTerminalPaddingIsTakenOut(t *testing.T) {
	got := Reply(verb.Out{Said: "ACME-3         to do                   the webhook retries"}, en())

	if strings.Contains(got, "   ") {
		t.Errorf("the padding survived: %q", got)
	}

	// Two spaces and not one: the columns are still columns, and "ACME-3 to
	// do" reads as three things where "ACME-3  to do" reads as two.
	if !strings.Contains(got, "ACME-3  to do  the webhook retries") {
		t.Errorf("it squeezed the columns away entirely: %q", got)
	}
}

// TestWhatFitsStaysATableAndWhatDoesNotGivesUpBeingOne.
//
// A monospaced block does not wrap, so past the width of a phone it stops
// buying anything: four wrapped lines a reader can see are worth more than
// one aligned line they have to drag sideways.
func TestWhatFitsStaysATableAndWhatDoesNotGivesUpBeingOne(t *testing.T) {
	narrow := Reply(verb.Out{Said: "agy  installed\nclaude  installed"}, en())
	if !strings.HasPrefix(narrow, "```") {
		t.Errorf("a listing that fits was not laid out: %q", narrow)
	}

	wide := Reply(verb.Out{Said: strings.Repeat("x", 60) + "  y\n" + "a  b"}, en())
	if strings.Contains(wide, "```") {
		t.Errorf("a listing too wide for a phone was still laid out: %q", wide)
	}
}

// TestAnAnswerTooLongToSendSaysSoRatherThanBeingRefused. A message one
// character over the service's ceiling is an answer nobody sees at all.
func TestAnAnswerTooLongToSendSaysSoRatherThanBeingRefused(t *testing.T) {
	got := Reply(verb.Out{Said: strings.Repeat("a", atMost*2)}, en())

	if len(got) > atMost+200 {
		t.Errorf("it answered with %d characters", len(got))
	}

	if !strings.Contains(got, "too long") {
		t.Errorf("it was cut without saying so: %q", got[len(got)-80:])
	}
}

// TestAFenceThatWasCutIsStillClosed. A code block that opened and never
// closed swallows whatever the service draws after it.
func TestAFenceThatWasCutIsStillClosed(t *testing.T) {
	// Narrow enough to be laid out, long enough to be cut.
	long := strings.TrimRight(strings.Repeat("ab  cd\n", atMost), "\n")

	got := Reply(verb.Out{Said: long}, en())
	if !strings.HasPrefix(got, "```") {
		t.Fatalf("a narrow listing was not laid out")
	}

	if strings.Count(got, "```") != 2 {
		t.Errorf("the fence was left open: %q", got[len(got)-60:])
	}
}

// TestNothingSaidIsStillAnAnswer. A verb that answered with nothing has done
// what it was asked, and a chat that said nothing back reads as one that
// broke.
func TestNothingSaidIsStillAnAnswer(t *testing.T) {
	if got := Reply(verb.Out{}, en()); got == "" {
		t.Error("a verb that said nothing produced no answer at all")
	}
}

var _ = words.For
