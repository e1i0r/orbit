package supervisor

// Every errand carries the rule that a promise is not an answer.

import (
	"strings"
	"testing"
)

// TestEveryErrandForbidsBackgroundingTheWork.
//
// The supervisor gets one turn. Three times running it put a long command
// into a background job and answered with the note its harness prints when
// it does that — "Wait for background task task-50 to complete", "I have
// launched `make check` ... and am waiting" — and the pull request it was
// asked for did not exist. internal/task/promise.go refuses those answers;
// this is the half that asks the engine not to give one.
//
// It is checked on the brief rather than on each body, because the brief
// is what every errand is wrapped in, and a rule stated five times is a
// rule four of them will eventually lose.
func TestEveryErrandForbidsBackgroundingTheWork(t *testing.T) {
	for _, want := range []string{
		"foreground",
		"background job",
		"not an answer",
	} {
		if !strings.Contains(strings.ToLower(supervisorBrief), want) {
			t.Errorf("the supervisor's brief does not say %q, so nothing tells an engine "+
				"to wait for the work it started", want)
		}
	}

	// And the brief reaches every body.
	for name, body := range map[string]string{
		"CreatePR":  CreatePR,
		"UpdatePR":  UpdatePR,
		"FixChecks": FixChecks,
		"MoreTests": MoreTests,
		"Review":    Review,
	} {
		if strings.TrimSpace(body) == "" {
			t.Errorf("%s has no instruction at all", name)
		}
	}
}

// TestCreatePRRunsTheChecksBeforeItOpensAnything.
//
// What the verb is for, in Elio's words: push the change, run the checks,
// and then create the pull request. The instruction named none of those
// middle steps, so the engine invented running `make check` — correctly —
// and then backgrounded it.
func TestCreatePRRunsTheChecksBeforeItOpensAnything(t *testing.T) {
	checks := strings.Index(CreatePR, "Run the repository's own checks")
	push := strings.Index(CreatePR, "Push the branch and open the pull request")
	url := strings.Index(CreatePR, "Answer with the URL")

	switch {
	case checks < 0:
		t.Fatal("CREATE PR never tells the supervisor to run the checks")
	case push < 0:
		t.Fatal("CREATE PR never tells the supervisor to push")
	case url < 0:
		t.Fatal("CREATE PR never asks for the URL")
	case checks >= push, push >= url:
		t.Errorf("CREATE PR asks for them out of order: checks at %d, push at %d, url at %d — "+
			"the checks come first, and the URL is the last thing", checks, push, url)
	}

	if !strings.Contains(CreatePR, "do not fix it") {
		t.Error("CREATE PR does not say to stop on a red tree; fixing the checks is its own verb")
	}
}
