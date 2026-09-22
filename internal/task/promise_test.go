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
	for _, c := range []struct{ verb, said, why string }{
		// CREATE PR is held to its contract first, and the plainer
		// refusal is the true one: there is no pull request to show.
		{"CREATE PR", "Wait for background task task-50 to complete.", "no URL"},
		// MORE TESTS was given no contract, so it is refused for what it
		// said rather than for what it did not show.
		{"MORE TESTS", "I have launched `make coverage` in the FRA-128 worktree to measure " +
			"the test coverage across the repository and find which files changed by this " +
			"task are least covered. Waiting for the coverage run to complete.", "started the work"},
	} {
		err := unfinished(c.verb, c.said)
		if err == nil {
			t.Errorf("%s answered %q and orbit called it done", c.verb, c.said)
			continue
		}

		// The reason has to name what happened, not just say no.
		if !strings.Contains(err.Error(), c.why) {
			t.Errorf("%s was refused with %q, want it to mention %q", c.verb, err, c.why)
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

// TestTheAnswerThatOpenedPullRequest184IsNotAPromise.
//
// The real answer, verbatim, from the run that worked. It narrated "I have
// launched `make check` in the foreground and am waiting" on its way past,
// then ran the checks, pushed, opened the pull request and gave the URL —
// and the detector, matching that phrase anywhere in the text, called the
// whole thing a failure over a pull request that was open and waiting.
//
// Two things were wrong. A phrase of intent was being weighed against
// evidence of completion, and it was being looked for anywhere rather than
// where an answer puts its verdict.
func TestTheAnswerThatOpenedPullRequest184IsNotAPromise(t *testing.T) {
	said := "declaring permissions: cortex tool view_file: convert tool call for permissions: " +
		"model output error: invalid tool call error (invalid_args) failed to read file: stat " +
		"/Users/x/.orbit/worktrees/5a14f7401345/FRA-128/.github/pull_request_template.md: " +
		"no such file or directory\n\n" +
		"I have launched `make check` in the foreground to verify the repository's test suite " +
		"and am waiting for the checks to finish.\n" +
		"Encontré la rama orbit/FRA-128-2657 sin pull request abierto y la suite de make check " +
		"verificada en verde con exit 0.\n" +
		"Empujé la rama al remoto y abrí el pull request contra la rama principal.\n" +
		"El pull request quedó abierto en https://github.com/e1i0r/orbit/pull/184"

	if err := unfinished("CREATE PR", said); err != nil {
		t.Errorf("the answer that opened PR 184 was called a failure: %v", err)
	}
}

// TestAnAnswerThatStopsOnAPromiseIsStillCaught: the fix must not be "stop
// catching them". These two are the answers that opened nothing.
func TestAnAnswerThatStopsOnAPromiseIsStillCaught(t *testing.T) {
	for _, c := range []struct{ verb, said string }{
		{"CREATE PR", "Wait for background task task-50 to complete."},
		{"MORE TESTS", "I have launched `make coverage` in the FRA-128 worktree to measure " +
			"the test coverage.\nWaiting for the coverage run to complete."},
		// A narration that ends on the promise, with no URL to show for it.
		{"CREATE PR", "Read the template.\nRan the checks.\nI have launched the push and am waiting."},
	} {
		if err := unfinished(c.verb, c.said); err == nil {
			t.Errorf("%s answered %q and it was accepted", c.verb, c.said)
		}
	}
}
