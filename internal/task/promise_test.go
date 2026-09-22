package task

// The two answers FRA-128 got, and the ones that must still pass.

import (
	"strings"
	"testing"
)

// TestThePromisesFRA128GotAreNotAnswers uses the exact text out of the
// record, because a detector written against a paraphrase of the failure
// is a detector that may not catch the failure.
func TestThePromisesFRA128GotAreNotAnswers(t *testing.T) {
	for _, c := range []struct{ verb, said string }{
		{"CREATE PR", "Wait for background task task-50 to complete."},
		{"MORE TESTS", "I have launched `make coverage` in the FRA-128 worktree to measure " +
			"the test coverage across the repository and find which files changed by this " +
			"task are least covered. Waiting for the coverage run to complete."},
	} {
		err := unfinished(c.verb, c.said)
		if err == nil {
			t.Errorf("%s answered %q and orbit called it done", c.verb, c.said)
			continue
		}

		// The reason has to name what happened, not just say no.
		if !strings.Contains(err.Error(), "started the work") {
			t.Errorf("%s was refused with %q, want it to say the work was only started", c.verb, err)
		}
	}
}

// TestAPullRequestVerbMustAnswerWithItsURL.
func TestAPullRequestVerbMustAnswerWithItsURL(t *testing.T) {
	for _, c := range []struct {
		verb, said string
		want       bool
		why        string
	}{
		{"CREATE PR", "https://github.com/e1i0r/orbit/pull/184", false, "a github url"},
		{
			"UPDATE PR", "Brought main in, nothing conflicted. https://github.com/e1i0r/orbit/pull/184",
			false, "a url in a sentence",
		},
		{"CREATE PR", "https://gitlab.com/acme/api/-/merge_requests/12", false, "gitlab"},
		{"CREATE PR", "https://bitbucket.org/acme/api/pull-requests/7", false, "bitbucket"},
		{"CREATE PR", "Opened the pull request.", true, "no url at all"},
		{
			"CREATE PR", "There is already one open for this branch, so I stopped.", true,
			"the step-1 refusal, which also has no url",
		},

		// The four that end in a report are held to nothing but the
		// promise rule: this has no way to grade prose.
		{
			"FIX CHECKS", "The lint job was failing on an unused import. Fixed, pushed, green.",
			false, "a report",
		},
		{"DEEP REVIEW", "Three things worth changing before this merges.", false, "a report"},

		// Silence is its own refusal, whatever the verb.
		{"FIX CHECKS", "   ", true, "nothing said"},
	} {
		err := unfinished(c.verb, c.said)
		if (err != nil) != c.want {
			t.Errorf("%s with %s (%q): refused = %v, want %v\n  said: %v",
				c.verb, c.why, c.said, err != nil, c.want, err)
		}
	}
}

// TestAnEnginesOwnErrorWins: a process that died is a better account than
// a rule about URLs, so Delivered keeps the first one.
func TestTheContractOnlyBindsTheVerbsThatWereGivenOne(t *testing.T) {
	// The same textless answer, held against a verb with a contract and
	// one without.
	if unfinished("CREATE PR", "done") == nil {
		t.Error("CREATE PR answered \"done\" with no URL and was accepted")
	}

	if err := unfinished("RESOLVE COMMENTS", "done"); err != nil {
		t.Errorf("RESOLVE COMMENTS was refused for having no URL, and it was never asked for one: %v", err)
	}
}
