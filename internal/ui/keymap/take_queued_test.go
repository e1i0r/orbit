package keymap

import (
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestTakeIsRefusedOnATaskWaitingInTheQueue. A task that ran before and is
// queued again has a session to take, and a run about to start in the same
// worktree: t took the old session beside it. It is refused, with the way
// out.
func TestTakeIsRefusedOnATaskWaitingInTheQueue(t *testing.T) {
	english, spanish := printers(t)

	queued := view.Task{
		ID: "ACME-12", Band: view.Running, Attempt: 1, Engine: "claude",
		Reason: view.Reason{Key: view.ReasonQueued},
	}

	why := whyNotTake(queued, Conditions{CanResume: true})
	if why.Name != whyTakeQueued {
		t.Fatalf("t on a queued task answers %q, want %q", why.Name, whyTakeQueued)
	}

	refused := Affordance{WhyNot: why}
	if en, es := refused.Why(english), refused.Why(spanish); en == es || en == "" {
		t.Errorf("the reason reads %q in English and %q in Spanish", en, es)
	}
}
