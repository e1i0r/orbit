package prompt

// What the window asks an engine for. These are instructions a model reads,
// so what is checked is that the facts the caller has — the task, the
// checkout, the engines this build owns — actually reach the page, and that
// the standing rules are still in front of them.

import (
	"encoding/json"
	"errors"
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
		got := Deliver("create PR", "ACME-1", "/checkouts/payments", body)

		for _, want := range []string{
			"ACME-1",              // which task
			"/checkouts/payments", // where to run
			"create PR",           // what the operator pressed
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
// keys that do the same, and the cockpit offers them as different verbs.
func TestEveryVerbAsksForSomethingDifferent(t *testing.T) {
	seen := map[string]bool{}
	for _, body := range []string{CreatePR, UpdatePR, FixChecks, MoreTests, Review, ResolveComments} {
		if seen[body] {
			t.Error("two of the deliver verbs ask for exactly the same thing")
		}

		seen[body] = true
	}
}

// TestTheDraftIsAskedForAsOneWholeDocument. A list of field names is
// something a model improvises around; one complete example is something it
// copies — so the example has to be valid JSON itself.
func TestTheDraftIsAskedForAsOneWholeDocument(t *testing.T) {
	got := FlowDraft("una revisión cuidadosa", []string{"zeta", "codex"})

	for _, want := range []string{
		"una revisión cuidadosa", // the request itself
		"zeta, codex",            // the engines this build has
		"nothing but the object", // the shape the answer takes
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the instruction does not mention %q", want)
		}
	}

	// The example is written on an engine this build owns, never on one it
	// does not: a phase naming an engine nobody has is a flow that cannot
	// run and a reader who has to work out why.
	if !strings.Contains(got, `"engine": "zeta"`) {
		t.Error("the example is not written on the first engine the build has")
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(example(t, got)), &doc); err != nil {
		t.Errorf("the example the model is told to copy is not valid JSON: %v", err)
	}

	if _, held := doc["phases"]; !held {
		t.Errorf("the example has no phases: %v", doc)
	}
}

// TestABuildWithNoEnginesStillHasAnExampleToShow, on the one every install
// starts with.
func TestABuildWithNoEnginesStillHasAnExampleToShow(t *testing.T) {
	if got := firstEngine(nil); got != "claude" {
		t.Errorf("a build with no engines writes the example on %q", got)
	}

	if got := firstEngine([]string{"codex"}); got != "codex" {
		t.Errorf("the example is written on %q, want the engine the build has", got)
	}
}

// TestTheSecondAskCarriesWhatWentWrongAndWhatWasWritten. The engine wrote
// it, so the engine is the one thing that knows what it meant to say.
func TestTheSecondAskCarriesWhatWentWrongAndWhatWasWritten(t *testing.T) {
	got := MendDraft(`{"name": "half a flow"`, errors.New("unexpected end of JSON input"))

	for _, want := range []string{"unexpected end of JSON input", "half a flow", "no fences"} {
		if !strings.Contains(got, want) {
			t.Errorf("the second ask does not mention %q", want)
		}
	}
}

// example is the JSON object out of the instruction: everything from the
// first brace to the last. It is bytes and not cells — the decoder reads
// bytes, and this is what will be handed to it.
func example(t *testing.T, said string) string {
	t.Helper()

	_, after, opened := strings.Cut(said, "{")
	if !opened {
		t.Fatal("the instruction carries no example at all")
	}

	closed := strings.LastIndex(after, "}")
	if closed < 0 {
		t.Fatal("the example in the instruction is never closed")
	}

	return "{" + after[:closed] + "}"
}
