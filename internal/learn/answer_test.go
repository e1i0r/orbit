package learn

// What a model wrote, read back: which of its lines become rules and which
// of them are quietly left where they are.

import (
	"strings"
	"testing"
)

// TestWhatCannotBeReadIsNotOffered.
//
// A model that explains itself first, bullets its answer, or invents a topic
// has not broken anything. What it wrote that cannot be read is simply not
// offered — failing over it would mean one badly-shaped answer costs the
// rules in the same reply that were fine.
func TestWhatCannotBeReadIsNotOffered(t *testing.T) {
	got := rulesIn(`Sure! Here is what I found:

- testing | anything that parses input gets fuzz tests
  vibes | the code should feel nicer
  style | comments explain, they do not judge
this line has no separator at all
  process |    `)

	if len(got) != 2 {
		t.Fatalf("the answer read as %d rules: %+v", len(got), got)
	}

	for _, one := range got {
		if !aTopic(one.topic) {
			t.Errorf("%q is not a topic anybody wrote down", one.topic)
		}
	}
}

// TestAModelCannotFillTheTrayFromOneHabit. Asked what four sentences have in
// common, a model can always find a fourth thing to say, and a tray full of
// weak proposals is a tray somebody stops opening.
func TestAModelCannotFillTheTrayFromOneHabit(t *testing.T) {
	var lines []string
	for range atMost + 4 {
		lines = append(lines, "testing | one more thing about the tests")
	}

	if got := rulesIn(strings.Join(lines, "\n")); len(got) != atMost {
		t.Errorf("one habit produced %d rules", len(got))
	}
}

// TestARuleAsLongAsOneMayBeIsStillARule.
//
// The length is what tells a rule from a paragraph, and a check one
// character too strict throws away the longest rule a model can write with
// nothing anywhere saying it did.
func TestARuleAsLongAsOneMayBeIsStillARule(t *testing.T) {
	longest := "testing | " + strings.Repeat("a", aRule)

	if got := rulesIn(longest); len(got) != 1 {
		t.Errorf("a rule of exactly %d characters read as %d rules", aRule, len(got))
	}

	if got := rulesIn(longest + "a"); len(got) != 0 {
		t.Errorf("a paragraph one character past %d was read as %d rules", aRule, len(got))
	}
}
