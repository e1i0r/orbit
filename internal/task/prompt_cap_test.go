package task

// What a prompt carries when Orbit knows more than one prompt can hold.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// rule is one rule at a scope, for a test that only cares about the shape.
func rule(phrase string, sc knowledge.Scope) knowledge.Rule {
	return knowledge.Rule{Phrase: phrase, Scope: sc, Source: knowledge.Human}
}

// gate is a rule a check enforces, which is what makes it stop the work.
func gate(phrase string, sc knowledge.Scope) knowledge.Rule {
	f := rule(phrase, sc)
	f.Stops, f.Check = true, "make check"

	return f
}

// general and deep are the widest and narrowest scopes these tests use.
func general() knowledge.Scope { return knowledge.Scope{Kind: knowledge.General} }

func deep(path string) knowledge.Scope {
	return knowledge.Scope{Kind: knowledge.Dir, Repo: "/r", Path: path}
}

// TestAShortListIsToldWhole, which is every checkout there is today.
func TestAShortListIsToldWhole(t *testing.T) {
	var facts []knowledge.Rule
	for range 5 {
		facts = append(facts, rule("something", general()))
	}

	kept, left := told(facts)
	if len(kept) != 5 || left != 0 {
		t.Errorf("a list of 5 was cut to %d with %d left over", len(kept), left)
	}
}

// TestALongListKeepsWhatStopsTheWork. A rule enforced by a gate is the one a
// phase cannot afford not to have seen: it is what sends the work back, and
// a phase that never read it walks into it.
func TestALongListKeepsWhatStopsTheWork(t *testing.T) {
	var facts []knowledge.Rule

	// Far more advice than fits, and one gate at the widest scope there is —
	// the last thing a narrowest-first rule would keep, if gates did not
	// come first.
	for i := range atMostTold * 2 {
		facts = append(facts, rule("advice "+string(rune('a'+i%26)), deep("a/b/c")))
	}

	facts = append(facts, gate("coverage stays above 90%", general()))

	kept, left := told(facts)
	if left == 0 {
		t.Fatal("nothing was cut, so this test asks nothing")
	}

	if !holds(kept, "coverage stays above 90%") {
		t.Error("the rule a gate enforces was cut, and the phase walks into the gate")
	}
}

// TestALongListKeepsWhatIsClosest. A rule about the directory being worked
// in is worth more to this phase than one about every project on the
// machine.
func TestALongListKeepsWhatIsClosest(t *testing.T) {
	var facts []knowledge.Rule

	for range atMostTold * 2 {
		facts = append(facts, rule("about everything", general()))
	}

	facts = append(facts, rule("about the ledger", deep("backend/ledger")))

	kept, left := told(facts)
	if left == 0 {
		t.Fatal("nothing was cut, so this test asks nothing")
	}

	if !holds(kept, "about the ledger") {
		t.Error("the rule closest to the code was cut in favour of rules about everything")
	}
}

// TestWhatIsKeptIsStillWidestFirst. The agent reads them in order, so the
// narrowest has to be last and have the last word — which is the opposite of
// the order they are chosen in.
func TestWhatIsKeptIsStillWidestFirst(t *testing.T) {
	var facts []knowledge.Rule

	for range atMostTold {
		facts = append(facts, rule("about the ledger", deep("backend/ledger")))
	}

	for range atMostTold {
		facts = append(facts, rule("about everything", general()))
	}

	kept, _ := told(facts)

	for i := 1; i < len(kept); i++ {
		if kept[i].Scope.Depth() < kept[i-1].Scope.Depth() {
			t.Fatalf("row %d is narrower than the one after it; the last word goes to the wrong rule", i-1)
		}
	}
}

// TestTheCutIsSaidInThePrompt. A prompt that quietly dropped half of what
// Orbit knows is one that claims to be the whole of it — an agent told the
// list is short can ask, and an agent told nothing cannot know there was
// anything to ask about.
func TestTheCutIsSaidInThePrompt(t *testing.T) {
	var facts []knowledge.Rule
	for range atMostTold + 7 {
		facts = append(facts, rule("something", general()))
	}

	said := whatIsKnown(facts)
	if !strings.Contains(said, "7 more rules are not in this prompt") {
		t.Errorf("the prompt does not say what it left out:\n%s", said)
	}

	// And a prompt that cut nothing says nothing about cutting.
	if whole := whatIsKnown(facts[:3]); strings.Contains(whole, "not in this prompt") {
		t.Errorf("a prompt that carried everything apologised for it anyway:\n%s", whole)
	}
}

// holds is whether a sentence is among the rules kept.
func holds(kept []knowledge.Rule, phrase string) bool {
	for _, f := range kept {
		if f.Phrase == phrase {
			return true
		}
	}

	return false
}
