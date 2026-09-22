package task

// Telling an answer from a promise to answer.
//
// A delivery verb is handed to an engine, and an engine can reply without
// having done the thing. FRA-128's CREATE PR came back with "Wait for
// background task task-50 to complete." — the agent had put the work in a
// background job and returned the note it prints when it does that. Orbit
// wrote that down as the answer, the tree drew a green tick, and no pull
// request existed. The branch was never even pushed.
//
// Nothing here can make an engine finish its work. What it can do is stop
// the record from calling a promise an answer, which is the difference
// between a reader who goes and looks and a reader who does not.

import (
	"errors"
	"regexp"
	"strings"
)

// wants is what a verb was told to answer with, by caption.
//
// The rule above this one applies to every verb there is and every verb
// added later: an engine that says it has begun has not finished, whatever
// it was asked to do. This map is the extra a verb earns by having been
// given an instruction that can be checked. CreatePR and UpdatePR both end
// "answer with the URL" (internal/supervisor/errands.go), so an answer
// with no link in it did not follow the last step it was given.
//
// The other four end in a report, and a report is prose: there is no way
// to tell a good one from a bad one here, and inventing a test for it
// would refuse honest answers to catch nothing. A verb given a checkable
// instruction later belongs on this map on the day it is given one.
var wants = map[string]*regexp.Regexp{
	"CREATE PR": prURL,
	"UPDATE PR": prURL,
}

// prURL is a link, and nothing is asked of its shape beyond that.
//
// It was /pull/, /pull-requests/ and /merge_requests/ at first, which is
// what the three forges spell. That is a rule about somebody else's URLs:
// a self-hosted forge, an enterprise host or a fourth service answers a
// different path, and refusing a real pull request for the shape of its
// link would be worse than the failure this is here to catch. A link is
// something a reader can click, and a promise never has one.
var prURL = regexp.MustCompile(`https?://\S+`)

// promises are the ways an engine says it has started something rather than
// done it.
//
// They are phrases and not single words, because the single words are all
// innocent in a real report: "waiting" is what a fix-checks answer says
// about a CI run it then watched to the end. What is being caught is the
// shape "I have begun, ask me later", which is the harness's own wording
// when a command is put in the background.
var promises = []string{
	"wait for background task",
	"waiting for background task",
	"wait for the background task",
	"i have launched",
	"i've launched",
	"i have started",
	"i've started",
	"waiting for the coverage run",
	"waiting for it to complete",
	"waiting for the run to complete",
	"will report back",
	"let me know when",
	"check back",
}

// unfinished is why what came back is not an answer, and nil when it is
// one.
//
// Silence is the plainest case: a verb that answered nothing answered
// nothing, whatever it may have done. Then the promises, which apply to
// every verb, and last the one contract a verb can be held to.
func unfinished(verb, text string) error {
	said := strings.TrimSpace(text)
	if said == "" {
		return errors.New("it came back having said nothing, so there is no account of what it did")
	}

	// The contract first, because it is evidence and the rest is reading
	// tone. A verb told to answer with the URL that answered with one did
	// the thing: whatever it said on the way there, the pull request is
	// open and a reader can click it.
	if want, held := wants[strings.ToUpper(strings.TrimSpace(verb))]; held {
		if want.MatchString(said) {
			return nil
		}

		return errors.New("it was asked to answer with the pull request's URL and there is " +
			"no URL in what it said, so there is nothing to show a reader")
	}

	// Then the tone, and only of the last thing it said.
	//
	// Anywhere in the answer was too blunt. CREATE PR ran the checks,
	// pushed, opened the pull request and reported the URL, and narrated
	// "I have launched `make check` in the foreground and am waiting" on
	// the way past — a true sentence about a step, in an answer that then
	// finished the job. What tells a promise from a report is where it
	// stops: an engine that ends on "I have begun" has not finished, and
	// one that ends on what happened has.
	low := strings.ToLower(endsOn(said))
	for _, p := range promises {
		if strings.Contains(low, p) {
			// Not quoted back. Whatever it said is written into the same
			// event and drawn one line under this on the task's tree, so
			// a quotation here is the answer twice, cut short the first
			// time.
			return errors.New("it said it had started the work and would finish later, " +
				"which is not the same as having done it")
		}
	}

	return nil
}

// endsOn is the final non-empty line of an answer, which is where an
// engine puts its verdict. A harness error or a progress note above it is
// not what the answer came to.
func endsOn(said string) string {
	lines := strings.Split(said, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}

	return said
}
