package panes

// A delivery verb that came back broken says how to ask again.

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// TestAFailedDeliveryVerbNamesTheKeyThatRetriesIt.
//
// The node went red with the reason on it and nothing else. Elio, looking
// at a failed CREATE PR: "I don't see the retry button here." There is no
// button — the gesture is the key it was offered under, which is true and
// which the screen never said.
func TestAFailedDeliveryVerbNamesTheKeyThatRetriesIt(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor", At: ago(2 * time.Minute)},
		{
			Kind: "deliver.answered", Verb: "CREATE PR", At: ago(time.Minute),
			Cause: "it said it had started the work and would finish later",
		},
	})

	steps := e.byHand()
	if len(steps) != 1 {
		t.Fatalf("the record holds %d hand steps, want one", len(steps))
	}

	drawn := ansi.Strip(strings.Join(subRows(e.handSubItems(steps[0]), "   "), "\n"))

	if !strings.Contains(drawn, "press p") {
		t.Errorf("a failed CREATE PR does not name the key that asks again:\n%s", drawn)
	}
}

// TestAVerbThatWorkedSaysNothingAboutRetrying.
func TestAVerbThatWorkedSaysNothingAboutRetrying(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor", At: ago(2 * time.Minute)},
		{
			Kind: "deliver.answered", Verb: "CREATE PR", At: ago(time.Minute),
			Text: "https://github.com/e1i0r/orbit/pull/184",
		},
	})

	drawn := ansi.Strip(strings.Join(subRows(e.handSubItems(e.byHand()[0]), "   "), "\n"))
	if strings.Contains(drawn, "again") {
		t.Errorf("a verb that worked offers a retry:\n%s", drawn)
	}
}

// TestEveryDeliveryVerbHasARetryKey: a caption with no key is a red node
// that says nothing, which is where this started.
func TestEveryDeliveryVerbHasARetryKey(t *testing.T) {
	for _, verb := range []string{
		"CREATE PR", "UPDATE PR", "FIX CHECKS", "MORE TESTS", "RESOLVE COMMENTS", "DEEP REVIEW",
	} {
		if _, ok := retryKey(verb); !ok {
			t.Errorf("%q has no key, so a reader whose %s failed is told nothing", verb, verb)
		}
	}

	// Merge and close are the operator's own keys and are not handed to
	// the supervisor, so they are not retried this way.
	if _, ok := retryKey("MERGE PR"); ok {
		t.Error("MERGE PR offers a supervisor retry, and it is the operator's own key")
	}
}
