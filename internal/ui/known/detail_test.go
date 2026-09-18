package known

// One rule, opened: what it says, where it stands, and what it is waiting
// for somebody to do about it.
//
// The three blocks that read almost nothing are the three a reader is
// actually deciding on: the band that says it is waiting on them, the reason
// it stopped applying, and the colour those are said in — which has to be
// the same colour wherever it is said, or a reader learns two vocabularies
// for one fact.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// opened is the screen with that rule read.
func opened(t *testing.T, f knowledge.Rule) (State, Env) {
	t.Helper()

	s, e := onScreen(t, f)
	e.Replace = func(knowledge.Rule, knowledge.Rule) error { return nil }

	s, _ = s.Key(press("enter"), e)
	if !s.reading {
		t.Fatal("enter did not open the rule")
	}

	return s, e
}

// TestARuleWaitingOnSomebodySaysSoAndWhichKeysAnswerIt. It is the board's own
// NEEDS YOU in the one other place it means the same thing, and a band that
// said a rule was waiting without saying what to press would be a reader
// hunting for a key.
func TestARuleWaitingOnSomebodySaysSoAndWhichKeysAnswerIt(t *testing.T) {
	s, e := opened(t, knowledge.Rule{
		ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human, Phrase: "the fuzz tests hang", Review: true,
	})

	drawn := drawnKnowledge(t, s, e)

	if !strings.Contains(drawn, "PENDING") {
		t.Errorf("a rule waiting on somebody does not say so:\n%s", drawn)
	}

	// The three answers, each named by the key that gives it.
	for _, way := range []string{"'c'", "'o'", "'u'"} {
		if !strings.Contains(drawn, way) {
			t.Errorf("the block does not offer %s:\n%s", way, drawn)
		}
	}
}

// TestARuleNobodyIsWaitingOnSaysNothingAboutIt. A band drawn over every rule
// is a band that means nothing, and the board's NEEDS YOU is worth
// something exactly because most rows are not in it.
func TestARuleNobodyIsWaitingOnSaysNothingAboutIt(t *testing.T) {
	s, e := opened(t, knowledge.Rule{
		ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human, Phrase: "the fuzz tests hang",
	})

	if drawn := drawnKnowledge(t, s, e); strings.Contains(drawn, "PENDING") {
		t.Errorf("a rule nobody is waiting on is drawn as one that is:\n%s", drawn)
	}
}

// TestWhyARulePausedIsInTheWordsOfWhoeverStoppedIt. A pause with no reason
// is a switch under another name, and the reason is what somebody reads when
// they come back — the only thing that will tell them whether it made sense.
func TestWhyARulePausedIsInTheWordsOfWhoeverStoppedIt(t *testing.T) {
	s, e := opened(t, knowledge.Rule{
		ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human, Phrase: "the fuzz tests hang",
		State: knowledge.Paused, Why: "the corpus is empty and it hangs for ever",
	})

	drawn := drawnKnowledge(t, s, e)

	if !strings.Contains(drawn, "the corpus is empty") {
		t.Errorf("the reason it was paused is not on the screen:\n%s", drawn)
	}

	// The section heads are drawn in capitals, so the reading is folded
	// rather than the sentence being written twice.
	if !strings.Contains(strings.ToLower(drawn), "not applying") {
		t.Errorf("the screen does not say the rule is not applying:\n%s", drawn)
	}
}

// TestARuleSwitchedOffWithNoReasonStillSaysWhatThatMeans. Nobody gives a
// reason for disagreeing with a rule — that is what switching it off is —
// so the block says what the state means and how to undo it, rather than
// drawing an empty quotation.
func TestARuleSwitchedOffWithNoReasonStillSaysWhatThatMeans(t *testing.T) {
	s, e := opened(t, knowledge.Rule{
		ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human, Phrase: "the fuzz tests hang",
		State: knowledge.Off,
	})

	drawn := drawnKnowledge(t, s, e)

	for _, want := range []string{"decided against it", "'u'"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the screen does not say %q:\n%s", want, drawn)
		}
	}
}

// TestARuleThatIsApplyingSaysNothingAboutWhyItIsNot, which is most of them.
func TestARuleThatIsApplyingSaysNothingAboutWhyItIsNot(t *testing.T) {
	s, e := opened(t, knowledge.Rule{
		ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human, Phrase: "the fuzz tests hang",
	})

	drawn := strings.ToLower(drawnKnowledge(t, s, e))
	if strings.Contains(drawn, "not applying") {
		t.Errorf("a rule that is applying says why it is not:\n%s", drawn)
	}
}

// TestAStateIsSaidInTheSameColourWhereverItIsSaid. The band in the list and
// the band on the rule are the same fact, and two colours for it is a reader
// learning the vocabulary twice.
func TestAStateIsSaidInTheSameColourWhereverItIsSaid(t *testing.T) {
	cases := []struct {
		name  string
		rule  knowledge.Rule
		band  band
		paint theme.Role
	}{
		{
			"waiting on somebody",
			knowledge.Rule{Source: knowledge.Human, Phrase: "x", Review: true},
			waits, theme.Warn,
		},
		{
			"it blocks the work",
			knowledge.Rule{Source: knowledge.Human, Phrase: "x", Check: "make check", Stops: true},
			stops, theme.Bad,
		},
		{
			"it only says its sentence",
			knowledge.Rule{Source: knowledge.Human, Phrase: "x"},
			says, theme.Live,
		},
		{
			"paused",
			knowledge.Rule{Source: knowledge.Human, Phrase: "x", State: knowledge.Paused, Why: "noisy"},
			paused, theme.Warn,
		},
		{
			"switched off",
			knowledge.Rule{Source: knowledge.Human, Phrase: "x", State: knowledge.Off},
			off, theme.Dim,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := bandOf(c.rule); got != c.band {
				t.Errorf("the rule is in band %v, want %v", got, c.band)
			}

			if got := bandRole(c.band); got != c.paint {
				t.Errorf("the band is painted %v, want %v", got, c.paint)
			}
		})
	}
}

// TestOnlyWhatIsWaitingOnAPersonIsWorthAColour. Five bands and one number
// that matters: how many things are waiting on somebody. A count lit on
// every band is five lit numbers and no signal.
func TestOnlyWhatIsWaitingOnAPersonIsWorthAColour(t *testing.T) {
	if got := bandCount(waits); got != theme.Warn {
		t.Errorf("the count of what is waiting is painted %v, want %v", got, theme.Warn)
	}

	for _, b := range []band{stops, says, paused, off} {
		if got := bandCount(b); got != theme.Dim {
			t.Errorf("the count of %v is painted %v, want it left alone", b, got)
		}
	}
}
