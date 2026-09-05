package task

// What Orbit knows, in the prompt. The rest of the prompt is prompt_test.go.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/knowledge"
)

func aFact(phrase string, sc knowledge.Scope) knowledge.Fact {
	return knowledge.Fact{Scope: sc, Source: knowledge.Human, Phrase: phrase}
}

// TestThePromptCarriesWhatOrbitKnows.
//
// This is how a fact reaches the model at all: Orbit already writes the
// prompt for every phase, so the sentences go in it. No hook in anybody's
// CLI, nothing to configure, and it arrives every time rather than when a
// model remembers to ask.
func TestThePromptCarriesWhatOrbitKnows(t *testing.T) {
	knows := []knowledge.Fact{
		aFact("The PRs and the commits are written in English.", knowledge.Scope{Kind: knowledge.General}),
		aFact("Never discard what a call answered with _.", knowledge.Scope{Kind: knowledge.Language, Lang: "go"}),
	}

	full := prompt(
		Task{ID: "ACME-1", Text: "Retry the webhook on 5xx.", Repo: repoTaskRepo(t)},
		flow.Phase{Name: "implement"}, knows, nil, "", nil,
	)

	if !strings.Contains(full, "## What Orbit knows") {
		t.Errorf("the prompt has no section for what is known:\n%s", full)
	}

	for _, said := range []string{"written in English", "Never discard what a call answered"} {
		if !strings.Contains(full, said) {
			t.Errorf("the prompt lost %q:\n%s", said, full)
		}
	}
}

// TestAFactThatStopsSaysSoInThePrompt. Being told a sentence and being told
// the gate will refuse the work over it are different things, and the model
// can only act on the difference if the prompt draws it.
func TestAFactThatStopsSaysSoInThePrompt(t *testing.T) {
	stops := aFact("No UPDATE or DELETE in ledger.", knowledge.Scope{Kind: knowledge.General})
	stops.Stops, stops.Check = true, "! git diff | grep -q 'UPDATE ledger'"

	full := prompt(
		Task{ID: "ACME-1", Text: "Fix the ledger.", Repo: repoTaskRepo(t)},
		flow.Phase{Name: "implement"}, []knowledge.Fact{stops}, nil, "", nil,
	)

	line := ""

	for _, l := range strings.Split(full, "\n") {
		if strings.Contains(l, "No UPDATE or DELETE") {
			line = l
		}
	}

	if !strings.Contains(strings.ToLower(line), "stop") {
		t.Errorf("a fact that stops the work reads like any other: %q", line)
	}
}

// TestWhatIsKnownIsReadBeforeWhatWasSaidNow. The facts are about the code and
// stand from before this task; a note is about this task. The narrower thing
// is read last so that it has the last word, which is the same order the
// scopes themselves are in.
func TestWhatIsKnownIsReadBeforeWhatWasSaidNow(t *testing.T) {
	full := prompt(
		Task{ID: "ACME-1", Text: "Retry the webhook.", Repo: repoTaskRepo(t)},
		flow.Phase{Name: "implement"},
		[]knowledge.Fact{aFact("The PRs are in English.", knowledge.Scope{Kind: knowledge.General})},
		[]string{"hold off on the backoff"}, "", nil,
	)

	knows, notes := strings.Index(full, "## What Orbit knows"), strings.Index(full, "## Operator notes")
	if knows < 0 || notes < 0 || knows > notes {
		t.Errorf("what is known is read after what was said now (%d and %d):\n%s", knows, notes, full)
	}
}

// TestNoFactsIsNoSection, the rule the rest of this prompt already follows:
// an empty heading is a question the model has to answer for itself.
func TestNoFactsIsNoSection(t *testing.T) {
	full := prompt(
		Task{ID: "ACME-1", Text: "Retry the webhook.", Repo: repoTaskRepo(t)},
		flow.Phase{Name: "implement"}, nil, nil, "", nil,
	)

	if strings.Contains(full, "## What Orbit knows") {
		t.Errorf("a prompt with nothing known still drew the heading:\n%s", full)
	}
}
