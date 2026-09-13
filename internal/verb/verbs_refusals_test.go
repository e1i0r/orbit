package verb

// What a verb does when the answer is no.
//
// Two ways it can be. The task it was asked about is not there, which is a
// typo and the commonest thing that happens to a command line. Or the
// machine underneath said no — the record would not open, the pull request
// would not go out, the terminal could not be handed over — and then the
// only wrong answer is to say it worked.

import (
	"errors"
	"strings"
	"testing"
)

// TestEveryVerbAboutATaskRefusesOneThatIsNotThere.
//
// Read off the declaration rather than listed here: a verb added tomorrow is
// held to this without anybody remembering to add it, which is the whole
// reason the vocabulary is declared in one place.
//
// The refusal has to name the task. "no such task" over a board of forty is
// a sentence that sends somebody to read forty rows to find out which one
// they mistyped.
func TestEveryVerbAboutATaskRefusesOneThatIsNotThere(t *testing.T) {
	w := worldOf(t)
	w.gitRepo(t, "acme")

	// task take is the one that does not look the task up: it hands the id
	// to the port and the port opens the checkout. Which means the refusal
	// lives in each way in rather than here — four chances to word it
	// differently, and the reason this package exists. Written down rather
	// than quietly skipped.
	elsewhere := map[string]string{
		"task take": "it hands the id to the port, and the port is what opens the checkout",
	}

	for _, v := range Every() {
		if !v.OnTask || elsewhere[v.Path()] != "" {
			continue
		}

		t.Run(v.Path(), func(t *testing.T) {
			err := refuseErr(t, w, v.Path(), In{Task: "NOPE-1", By: "operator", Args: filled(v)})
			if !strings.Contains(err.Error(), "NOPE-1") {
				t.Errorf("refused with %q, which names no task", err)
			}
		})
	}
}

// TestAVerbAnswersTheFailureOfWhatItReachedFor.
//
// The ports are the machine: the record, the pull request, the terminal, the
// file the export writes. A verb that swallowed one of their refusals would
// answer "done" about work that did not happen, and the reader would find
// out at the next thing that depended on it.
func TestAVerbAnswersTheFailureOfWhatItReachedFor(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-20", r.Path, "pay the thing")

	// Set after the task is written down, so what is being refused is the
	// port and not the writing.
	w.refuse = errors.New("the machine said no")

	for _, one := range []struct {
		verb string
		in   In
	}{
		{"pr", In{Task: "ACME-20", Repo: r.Path}},
		{"pr merge", In{Task: "ACME-20", Repo: r.Path}},
		{"pr close", In{Task: "ACME-20", Repo: r.Path}},
		{"task take", In{Task: "ACME-20", Repo: r.Path}},
		{"task show", In{Task: "ACME-20", Repo: r.Path}},
		{"knowledge", In{}},
		{"knowledge learn", In{Args: map[string]string{"text": "amounts are cents", "repo": r.Path}}},
		{"supervisor say", In{Args: map[string]string{"text": "never force-push"}}},
		{"export", In{Args: map[string]string{"into": t.TempDir() + "/out"}}},
	} {
		t.Run(one.verb, func(t *testing.T) {
			one.in.By = "operator"

			if err := refuseErr(t, w, one.verb, one.in); !strings.Contains(err.Error(), "said no") {
				t.Errorf("answered %q, and the machine's own refusal is not in it", err)
			}
		})
	}
}

// filled is the fields a verb must have, with something in them, so that the
// refusal under test is the missing task and not a missing argument.
func filled(v Verb) map[string]string {
	args := map[string]string{}

	for _, f := range v.Takes {
		if f.Needed {
			args[f.Name] = "1"
		}
	}

	return args
}
