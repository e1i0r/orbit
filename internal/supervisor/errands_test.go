package supervisor

// What the deliver verbs hand the supervisor. These are instructions a
// model reads, so what is checked is that the facts the caller has — the
// task, the checkout, where the ask came from — actually reach the page,
// and that the standing rules are still in front of them.

import (
	"strings"
	"testing"
)

// TestEveryDeliverVerbCarriesTheBriefAndTheFactsItNeeds.
//
// The brief is where the rules live — never force-push, do not merge, say so
// if this is the wrong thing to do — and a body sent without it is a model
// asked to do something with no idea what it may not do.
func TestEveryDeliverVerbCarriesTheBriefAndTheFactsItNeeds(t *testing.T) {
	for _, body := range []string{CreatePR, UpdatePR, FixChecks, MoreTests, Review, ResolveComments} {
		got := Deliver("the cockpit", "create PR", "ACME-1", "/checkouts/payments", body)

		for _, want := range []string{
			"ACME-1",              // which task
			"/checkouts/payments", // where to run
			"create PR",           // what the operator asked for
			"the cockpit",         // where they asked it
			"Never force-push",    // the rule that matters most
			"Do not merge",        // the operator's own keys
			"in English",          // what the repository is written in
		} {
			if !strings.Contains(got, want) {
				t.Errorf("the instruction does not mention %q", want)
			}
		}

		if !strings.HasSuffix(got, body) {
			t.Error("the verb's own body is not at the end of the brief")
		}
	}
}

// TestEveryVerbAsksForSomethingDifferent. Two that read the same are two
// keys that do the same, and the surfaces offer them as different verbs.
func TestEveryVerbAsksForSomethingDifferent(t *testing.T) {
	seen := map[string]bool{}
	for _, body := range []string{CreatePR, UpdatePR, FixChecks, MoreTests, Review, ResolveComments} {
		if seen[body] {
			t.Error("two of the deliver verbs ask for exactly the same thing")
		}

		seen[body] = true
	}
}
