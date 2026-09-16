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

// TestOnlyWhatStopsARunIsWorthAMessage. A channel that tells you everything
// is a channel you mute, and a muted channel is worse than none — you
// believed you would be told.
func TestOnlyWhatStopsARunIsWorthAMessage(t *testing.T) {
	for _, kind := range []string{
		record.PhaseStarted, record.PhaseToolCall, record.PhaseThought,
		record.PhaseFinished, record.TaskStarted, record.PhaseAsked,
		record.GatePassed, record.TaskRelayed,
	} {
		if said := News(happened("ACME-1", kind, "implement", nil), en()); said != "" {
			t.Errorf("%s was worth a message: %q", kind, said)
		}
	}

	for _, kind := range []string{
		record.PhaseWaiting, record.TaskStuck, record.TaskNeedsEngine,
		record.TaskNoEngine, record.TaskFinished,
	} {
		said := News(happened("ACME-1", kind, "implement", map[string]string{
			"attempts": "3", "from": "claude", "engines": "codex,opencode",
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
