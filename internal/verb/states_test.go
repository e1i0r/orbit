package verb

// Where the rules stand, listed and moved.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// aRule is one rule of a checkout, under a name of its own.
func aRule(id, phrase string) knowledge.Fact {
	return knowledge.Fact{
		ID: id, Source: knowledge.Human, Phrase: phrase,
		Scope: knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/acme"},
	}
}

// TestOneListingAnsweringOneQuestion.
//
// With nothing asked it is what wants a decision from you — the sentences
// nobody has answered, and the rules sent to be looked at again — because
// that is what somebody who has just sat down came for. Two screens would be
// two to remember to look at.
func TestOneListingAnsweringOneQuestion(t *testing.T) {
	w := trayOf(t, "never push without the tests passing")

	again := aRule("aaaa1111", "coverage stays above 90%")
	again.State, again.Why, again.Review = knowledge.Paused, "until we are past 80", true
	w.facts = []knowledge.Fact{again, aRule("bbbb2222", "amounts are cents")}

	out, err := asked(t, w, "rules", nil)
	if err != nil {
		t.Fatalf("rules: %v", err)
	}

	for _, want := range []string{"never push without", "coverage stays above", "until we are past 80"} {
		if !strings.Contains(out.Said, want) {
			t.Errorf("what needs an answer does not say %q:\n%s", want, out.Said)
		}
	}

	// And nothing that is not waiting for one.
	if strings.Contains(out.Said, "amounts are cents") {
		t.Errorf("a rule nobody has to decide about is in the listing:\n%s", out.Said)
	}
}

// TestTheListingFiltersByWhereARuleStands, which is the other half: what is
// working, what is stopped, what was thrown out.
func TestTheListingFiltersByWhereARuleStands(t *testing.T) {
	w := trayOf(t)

	paused := aRule("aaaa1111", "coverage stays above 90%")
	paused.State = knowledge.Paused

	off := aRule("cccc3333", "never push on a Friday")
	off.State = knowledge.Off

	w.facts = []knowledge.Fact{paused, off, aRule("bbbb2222", "amounts are cents")}

	for _, one := range []struct{ state, want, gone string }{
		{"active", "amounts are cents", "coverage stays"},
		{"paused", "coverage stays", "amounts are cents"},
		{"off", "never push on a Friday", "amounts are cents"},
	} {
		t.Run(one.state, func(t *testing.T) {
			out, err := asked(t, w, "rules", map[string]string{"state": one.state})
			if err != nil {
				t.Fatalf("rules -state %s: %v", one.state, err)
			}

			if !strings.Contains(out.Said, one.want) {
				t.Errorf("it does not say %q:\n%s", one.want, out.Said)
			}

			if strings.Contains(out.Said, one.gone) {
				t.Errorf("it says %q, which stands somewhere else:\n%s", one.gone, out.Said)
			}
		})
	}
}

// TestAStateNobodyCanBeInIsRefusedInWords, because a filter that quietly
// answered nothing would read as "there are none of those" about a question
// that was never asked.
func TestAStateNobodyCanBeInIsRefusedInWords(t *testing.T) {
	w := trayOf(t)

	_, err := asked(t, w, "rules", map[string]string{"state": "dormant"})
	if err == nil {
		t.Fatal("a state nobody can be in was accepted")
	}

	if !strings.Contains(err.Error(), "dormant") {
		t.Errorf("the refusal reads %q, without saying what was typed", err)
	}
}

// TestPausingTakesAReasonAndAsksAboutItLater.
//
// The reason is the whole difference between this and a switch: it is what
// somebody reads when they come back, and the only thing that will tell them
// whether it made sense or they were wrong.
func TestPausingTakesAReasonAndAsksAboutItLater(t *testing.T) {
	w := trayOf(t)
	w.facts = []knowledge.Fact{aRule("aaaa1111", "coverage stays above 90%")}

	out, err := asked(t, w, "rules pause", map[string]string{
		"rule": "aaaa1111", "why": "the repo has never been past 80",
	})
	if err != nil {
		t.Fatalf("pause: %v", err)
	}

	if !strings.Contains(out.Said, "never been past 80") {
		t.Errorf("pausing answered %q, which does not say what for", out.Said)
	}

	now := w.facts[0]
	if now.State != knowledge.Paused || now.Why == "" {
		t.Errorf("the rule stands at %v, for %q", now.State, now.Why)
	}

	if !now.Review {
		t.Error("a paused rule is not waiting for a decision")
	}

	if now.Tells() {
		t.Error("a paused rule still reaches a phase")
	}
}

// TestResumingIsTheAnswerToBothWaysItGotThere: paused on purpose, or skipped
// once and left waiting.
func TestResumingIsTheAnswerToBothWaysItGotThere(t *testing.T) {
	w := trayOf(t)

	was := aRule("aaaa1111", "coverage stays above 90%")
	was.State, was.Why, was.Review = knowledge.Paused, "until we are past 80", true
	w.facts = []knowledge.Fact{was}

	if _, err := asked(t, w, "rules resume", map[string]string{"rule": "aaaa1111"}); err != nil {
		t.Fatalf("resume: %v", err)
	}

	now := w.facts[0]
	if !now.Tells() || now.Review || now.Why != "" {
		t.Errorf("the rule stands at %v, review=%v, why=%q", now.State, now.Review, now.Why)
	}
}

// TestARuleNobodyHasNamedCannotBeReachedByName.
//
// A rule somebody wrote by hand has no name until Orbit writes it, so there
// is nothing to type — and the refusal says where the names are printed
// rather than only that this one was not found.
func TestARuleNobodyHasNamedCannotBeReachedByName(t *testing.T) {
	w := trayOf(t)
	w.facts = []knowledge.Fact{aRule("", "written by hand, with no name")}

	_, err := asked(t, w, "rules pause", map[string]string{"rule": "aaaa1111", "why": "because"})
	if err == nil {
		t.Fatal("a rule with no name was reached by naming another")
	}

	if !strings.Contains(err.Error(), "orbit knowledge") {
		t.Errorf("the refusal reads %q, without saying where the names are", err)
	}
}
