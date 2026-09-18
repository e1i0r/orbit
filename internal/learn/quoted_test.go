package learn

// What the model is shown about one file, and what is read back out of its
// answer.

import (
	"strings"
	"testing"
)

// TestWithNoHistoryTheModelIsShownOnlyTheFile.
//
// What the commits say the project actually does is measured before this is
// called, and on a checkout with no history there is nothing to say. An
// empty heading followed by no lines reads to a model as "the history says
// this project does nothing", and the sentence under it asks the model to
// leave out everything the file contradicts.
func TestWithNoHistoryTheModelIsShownOnlyTheFile(t *testing.T) {
	alone := aboutThisProject("CONTRIBUTING.md", aContributing, atMostCold, nil)

	if strings.Contains(alone, "actually does") {
		t.Error("a checkout with no history was shown a heading with nothing under it")
	}

	// And where there is one, it arrives with the rule about what to do with
	// it: a sentence nobody has held to for a year is not a rule.
	with := aboutThisProject("CONTRIBUTING.md", aContributing, atMostCold,
		[]string{"nobody has run the fuzz tests in a year"})

	for _, want := range []string{"actually does", "nobody has run the fuzz tests in a year"} {
		if !strings.Contains(with, want) {
			t.Errorf("the model was not shown %q", want)
		}
	}
}

// TestTheReadingStopsAtTheRoomItWasGiven.
//
// The cap is on the whole reading and the model is only asked to keep to it.
// A model that writes ten lines anyway has not broken anything as long as
// what is read back stops where it was told to — and a reading that stopped
// at the first line instead would spend a call on a file to take one rule
// out of it.
func TestTheReadingStopsAtTheRoomItWasGiven(t *testing.T) {
	answer := anAnswerOf(len(theSentencesOf))

	if got := quoted(answer, aRuled, atMostCold); len(got) != atMostCold {
		t.Errorf("six rules read with room for %d came back as %d", atMostCold, len(got))
	}

	if got := quoted(answer, aRuled, len(theSentencesOf)); len(got) != len(theSentencesOf) {
		t.Errorf("six rules read with room for six came back as %d", len(got))
	}
}

// TestARuleFromAFileAsLongAsOneMayBeIsKept.
//
// The length is what tells a rule from a paragraph copied out of a document,
// and a check one character too strict throws away the longest rule a model
// can write with nothing anywhere saying it did.
func TestARuleFromAFileAsLongAsOneMayBeIsKept(t *testing.T) {
	quote := theSentencesOf[0]
	longest := strings.Repeat("a", aRule)

	if _, ok := aRuleFrom("testing", longest, quote, aRuled); !ok {
		t.Errorf("a rule of exactly %d characters was thrown away", aRule)
	}

	if _, ok := aRuleFrom("testing", longest+"a", quote, aRuled); ok {
		t.Errorf("a paragraph one character past %d was kept as a rule", aRule)
	}
}

// TestARuleWithoutItsLineIsNotKept, which is the whole reason a reading of a
// document can be trusted: a model asked to summarise two years of
// CONTRIBUTING will produce plausible rules nobody ever wrote.
func TestARuleWithoutItsLineIsNotKept(t *testing.T) {
	for _, one := range []struct {
		why    string
		topic  string
		phrase string
		quote  string
	}{
		{"a sentence that is not in the file", "testing", "run the fuzz tests", "Run the fuzz tests."},
		{"a topic nobody wrote down", "vibes", "make it nicer", theSentencesOf[0]},
		{"no rule at all", "testing", "", theSentencesOf[0]},
		{"no line to point at", "testing", "run the fuzz tests", ""},
	} {
		if _, ok := aRuleFrom(one.topic, one.phrase, one.quote, aRuled); ok {
			t.Errorf("%s was kept as a rule", one.why)
		}
	}
}
