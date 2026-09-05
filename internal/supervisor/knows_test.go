package supervisor

// What the supervisor is told it already knows, and what it does about
// being told something it should have known.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// TestTheSupervisorIsToldWhatOrbitAlreadyKnows.
//
// It answers questions about the work and directs tasks, and it was doing
// both without the standing rules in front of it — so it could say something
// a gate would refuse an hour later, and could be told a rule it had already
// been given without noticing.
func TestTheSupervisorIsToldWhatOrbitAlreadyKnows(t *testing.T) {
	asked := buildSupervisorPrompt("", "what should I look at?", []knowledge.Fact{
		{Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "the PRs are written in English"},
	})

	if !strings.Contains(asked, "the PRs are written in English") {
		t.Errorf("the supervisor is not told what Orbit knows:\n%s", asked)
	}
}

// TestKnowingNothingDrawsNoSection, the rule every other part of this prompt
// follows: an empty heading is a question the model answers for itself.
func TestKnowingNothingDrawsNoSection(t *testing.T) {
	asked := buildSupervisorPrompt("", "what should I look at?", nil)
	if strings.Contains(asked, "What Orbit already knows") {
		t.Errorf("a heading was drawn with nothing under it:\n%s", asked)
	}
}

// TestTheSupervisorIsAskedToOfferToRememberThings.
//
// The eje is not "you said it twice": it is being told something that should
// have been standing. A model reading the thread and the facts can see that,
// where matching text cannot — the same thing said in different words is not
// the same string.
func TestTheSupervisorIsAskedToOfferToRememberThings(t *testing.T) {
	asked := strings.ToLower(buildSupervisorPrompt("", "the fuzz tests hang sometimes", nil))

	for _, want := range []string{"orbit_learn", "offer"} {
		if !strings.Contains(asked, want) {
			t.Errorf("the supervisor is not asked to offer to write things down (%q missing):\n%s", want, asked)
		}
	}
}

// TestTheSupervisorIsToldNotToWriteWithoutBeingAsked. Nothing is promoted
// silently: a rule that appeared without somebody agreeing to it is a rule
// nobody put there, and the first one of those costs the whole feature its
// trust.
func TestTheSupervisorIsToldNotToWriteWithoutBeingAsked(t *testing.T) {
	asked := strings.ToLower(buildSupervisorPrompt("", "anything", nil))
	if !strings.Contains(asked, "without") || !strings.Contains(asked, "agree") {
		t.Errorf("nothing tells the supervisor to ask first:\n%s", asked)
	}
}
