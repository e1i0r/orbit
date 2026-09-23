package task

// How many times the supervisor may send one task round again.

import (
	"context"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
)

// TestTheSupervisorCannotSendATaskRoundForEver.
//
// A verdict of `again` puts the task back in To Do, and autopilot starts
// what is in To Do: the same run, judged by the same model, which says
// `again` for the same reason. Nothing counted them, so a task the
// validator was never going to be happy with ran for as long as the
// machine was on, at a full run's cost each time round.
//
// Three runs is the cap — the first and two more — after which it is a
// person's to look at, which is the other answer the validator already
// has.
func TestTheSupervisorCannotSendATaskRoundForEver(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-45", "something the judge never likes", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := &judgeEngine{Fake: engine.NewFake("did the work"), verdict: "verdict: again still missing the tests"}

	// Six runs of the same task, as autopilot would: each one picks the
	// task up out of To Do and walks the flow again.
	for range 6 {
		if err := Run(context.Background(), s, tk, validatedFlow(true), fakes(eng), nil); err != nil {
			t.Fatalf("Run: %v", err)
		}
	}

	events := mustEvents(t, s, tk)

	sent := 0

	for _, e := range events {
		if e.Kind == record.TaskRequeued && e.Data["by"] == "supervisor" {
			sent++
		}
	}

	if sent > supervisorRuns {
		t.Errorf("the supervisor sent the task round %d times, and the cap is %d: %v",
			sent, supervisorRuns, kindsOf(events))
	}

	// And what it ends as is a person's decision rather than another run.
	last := events[len(events)-1]
	if last.Kind != record.TaskStuck {
		t.Errorf("the record ends in %q, want task.stuck once the supervisor has run out of tries: %v",
			last.Kind, kindsOf(events))
	}
}
