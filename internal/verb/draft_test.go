package verb

// What somebody keeps telling runs, and the one reading that costs money.

import (
	"errors"
	"strings"
	"testing"
)

// errAsked is the machine saying no, for the one port that reaches a model.
var errAsked = errors.New("no engine on this machine")

// keptSaying writes a task and tells it the same thing three times, which is
// what turns a correction into a habit.
func keptSaying(t *testing.T, w *testWorld, id string, said ...string) {
	t.Helper()

	r := w.gitRepo(t, "acme")
	w.wrote(t, id, r.Path, "pay the thing")

	for _, one := range said {
		mustAsk(t, w, "task direct",
			In{Task: id, Repo: r.Path, Args: map[string]string{"text": one}, By: "operator"})
	}
}

// TestWhatYouRepeatIsListedWithTheSentencesItCameFrom.
//
// The words somebody used are the rule, and the number of times is only the
// reason to go and look at them. A listing that gave the count alone would
// be asking them to remember what they said.
func TestWhatYouRepeatIsListedWithTheSentencesItCameFrom(t *testing.T) {
	w := worldOf(t)
	keptSaying(t, w, "ACME-50",
		"add fuzz testing to this",
		"put some fuzz tests on the parser",
		"this needs fuzz tests too")

	out := mustAsk(t, w, "rules repeated", In{By: "operator"})

	for _, want := range []string{"fuzz", "ACME-50", "add fuzz testing to this"} {
		if !strings.Contains(out.Said, want) {
			t.Errorf("the listing does not say %q:\n%s", want, out.Said)
		}
	}
}

// TestTheModelsRuleWaitsInTheTrayLikeAnyOther.
//
// This is the one reading in Orbit that costs money and the one place a
// model decides anything, and what it produces is still only an offer: it
// waits to be kept or dropped exactly as a sentence somebody typed does.
func TestTheModelsRuleWaitsInTheTrayLikeAnyOther(t *testing.T) {
	w := worldOf(t)
	w.answer = "testing | anything that parses input gets fuzz tests"

	keptSaying(t, w, "ACME-51",
		"add fuzz testing to this",
		"put some fuzz tests on the parser",
		"this needs fuzz tests too")

	out := mustAsk(t, w, "rules draft", In{By: "operator"})

	if !strings.Contains(out.Said, "fuzz tests") || !strings.Contains(out.Said, "testing") {
		t.Errorf("drafting answered %q", out.Said)
	}

	// Said out loud rather than left to be discovered: a reader told a rule
	// was written would believe the next run is already told it.
	if !strings.Contains(out.Said, "waiting") {
		t.Errorf("drafting answered %q, which does not say it is not in effect", out.Said)
	}

	// And the model was shown the sentences, which is the whole of what it
	// is given: not the code, not the diff, not the record.
	if !strings.Contains(w.asked, "put some fuzz tests on the parser") {
		t.Errorf("the model was asked:\n%s", w.asked)
	}

	tray := mustAsk(t, w, "rules", In{By: "operator"})
	if !strings.Contains(tray.Said, "anything that parses input") {
		t.Errorf("the rule is not in the tray:\n%s", tray.Said)
	}
}

// TestWithNothingRepeatedThereIsNothingToDraft, and it says so rather than
// saying nothing: a command that answers with an empty screen reads as one
// that failed quietly.
func TestWithNothingRepeatedThereIsNothingToDraft(t *testing.T) {
	w := worldOf(t)
	w.answer = "testing | this should never be reached"

	for _, one := range []string{"rules repeated", "rules draft"} {
		out := mustAsk(t, w, one, In{By: "operator"})
		if out.Said == "" {
			t.Errorf("%q answered with nothing at all", one)
		}
	}
}

// TestAnEngineThatWillNotRunIsSaidPlainly. This is the one path that needs a
// model, and a machine without one has to be told so rather than shown a
// listing that is empty for a reason nobody can see.
func TestAnEngineThatWillNotRunIsSaidPlainly(t *testing.T) {
	w := worldOf(t)
	keptSaying(t, w, "ACME-52",
		"add fuzz testing to this",
		"put some fuzz tests on the parser",
		"this needs fuzz tests too")

	w.refuse = errAsked

	if err := refuseErr(t, w, "rules draft", In{By: "operator"}); err == nil {
		t.Error("an engine that would not answer was reported as though it had")
	}
}
