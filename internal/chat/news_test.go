package chat

// What is worth interrupting somebody for, and what it says.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

func happened(task, kind, phase string, data map[string]string) Happening {
	return Happening{Task: task, Event: record.Event{Kind: kind, Phase: phase, Data: data}}
}

// TestWhatHappensInsideAPhaseIsNotNews. The line is what changes where a
// task is; everything inside a phase is a run working rather than a run
// moving, and there are hundreds of them. A channel that carried those is a
// channel you mute, which is worse than none — you believed you would be
// told.
func TestWhatHappensInsideAPhaseIsNotNews(t *testing.T) {
	for _, kind := range []string{
		record.PhaseStarted, record.PhaseToolCall, record.PhaseThought,
		record.PhaseFinished, record.PhaseAsked, record.PhaseRetried,
		record.GatePassed, record.TaskNoted, record.TaskRead, record.TaskCreated,
	} {
		if said := News(happened("ACME-1", kind, "implement", nil), en()); said != "" {
			t.Errorf("%s was worth a message: %q", kind, said)
		}
	}

	// And the arc of a task is: it began, it changed hands, it stopped, it
	// ended. Somebody away from their desk can follow that.
	for _, kind := range []string{
		record.TaskStarted, record.TaskRelayed,
		record.PhaseWaiting, record.TaskStuck, record.TaskNeedsEngine,
		record.TaskNoEngine, record.TaskFailed, record.TaskOverBudget,
		record.TaskOverDiff, record.TaskNewDependency, record.TaskContradicts,
		record.TaskFinished, record.TaskMerged, record.TaskCancelled,
		record.TaskTimedOut, record.TaskAbandoned,
	} {
		said := News(happened("ACME-1", kind, "implement", map[string]string{
			"attempts": "3", "from": "claude", "to": "codex", "engines": "codex,opencode",
			"spent": "$2", "budget": "$1", "lines": "900", "names": "left-pad",
			"decision": "REF-9",
		}), en())

		if said == "" {
			t.Errorf("%s was not worth a message", kind)
		}

		if !strings.Contains(said, "ACME-1") {
			t.Errorf("%s does not say which task: %q", kind, said)
		}
	}
}

// TestAMessageCarriesWhatToDoAboutIt. One that does not is a phone opened
// for nothing, and it is how a reader learns to ignore the next one.
func TestAMessageCarriesWhatToDoAboutIt(t *testing.T) {
	needs := News(happened("ACME-3", record.TaskNeedsEngine, "implement", map[string]string{
		"from": "claude", "engines": "codex,opencode",
	}), en())

	// The engines that could take it, and one command to tap rather than a
	// choice between three.
	if !strings.Contains(needs, "codex, opencode") {
		t.Errorf("it does not name who could take it: %q", needs)
	}

	if !strings.Contains(needs, "/task start ACME-3 -engine codex") {
		t.Errorf("it does not offer the command that answers it: %q", needs)
	}

	waiting := News(happened("ACME-3", record.PhaseWaiting, "implement", nil), en())
	if !strings.Contains(waiting, "/task continue ACME-3") {
		t.Errorf("a gate does not offer the word that lets it go: %q", waiting)
	}
}

// TestTheOneWithNothingToOfferOffersNothing. No engine has anything left, so
// there is no command that would change it — and a command that cannot work
// is worse than none.
func TestTheOneWithNothingToOfferOffersNothing(t *testing.T) {
	said := News(happened("ACME-3", record.TaskNoEngine, "implement", map[string]string{
		"from": "claude", "back": "1h30m0s",
	}), en())

	if strings.Contains(said, "/task start") {
		t.Errorf("it offered a command that cannot work: %q", said)
	}

	if !strings.Contains(said, "claude") {
		t.Errorf("it does not say which engine ran out: %q", said)
	}
}

// TestAPullRequestIsNewsAndKeepingOneUpToDateIsNot. Opening one is where a
// task stops being Orbit's and starts being the team's, which is the one
// thing here somebody else will see.
func TestAPullRequestIsNewsAndKeepingOneUpToDateIsNot(t *testing.T) {
	opened := News(happened("ACME-3", record.DeliverAnswered, "", map[string]string{"verb": "pr"}), en())
	if !strings.Contains(opened, "pull request is open") {
		t.Errorf("a pull request opening was not news: %q", opened)
	}

	for _, verb := range []string{"update", "checks", "resolve", "tests", "review"} {
		said := News(happened("ACME-3", record.DeliverAnswered, "", map[string]string{"verb": verb}), en())
		if said != "" {
			t.Errorf("%s was news: %q", verb, said)
		}
	}

	// And one that did not open says so, with why.
	broke := News(happened("ACME-3", record.DeliverAnswered, "", map[string]string{
		"verb": "pr", "error": "no upstream\nand more",
	}), en())

	if !strings.Contains(broke, "did not open") || !strings.Contains(broke, "no upstream") {
		t.Errorf("a pull request that broke says %q", broke)
	}

	// The first line of it and no more: a notification is one line somebody
	// reads on the way past.
	if strings.Contains(broke, "and more") {
		t.Errorf("it carried the whole failure: %q", broke)
	}
}
