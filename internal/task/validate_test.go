package task

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// judgeEngine does the work, and then says one thing about it when it is
// asked to judge.
type judgeEngine struct {
	*engine.Fake

	verdict string
}

func (e *judgeEngine) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	if strings.Contains(req.Prompt, validateMarker) {
		return engine.Result{Output: e.verdict}, nil
	}

	return e.Fake.Run(ctx, req)
}

func validatedFlow(on bool) flow.Flow {
	return flow.Flow{Name: "task", Validate_: on, Phases: []flow.Phase{
		{Name: "implement", Engine: "fake"},
	}}
}

func ranWith(t *testing.T, id, verdict string, on bool) ([]string, []record.Event) {
	t.Helper()

	s, r := fixture(t)

	tk, err := Create(s, r, id, "make the endpoint idempotent", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := &judgeEngine{Fake: engine.NewFake("did the work"), verdict: verdict}
	if err := Run(context.Background(), s, tk, validatedFlow(on), map[string]engine.Engine{"fake": eng}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events := mustEvents(t, s, tk)

	return kindsOf(events), events
}

// TestAVerdictOfDoneLetsTheRunFinish. Prefer done: a validator that sent
// good work round again would cost a run every time it disagreed about
// style.
func TestAVerdictOfDoneLetsTheRunFinish(t *testing.T) {
	kinds, _ := ranWith(t, "ACME-41", "verdict: done\nIt answers the task.", true)

	if count(kinds, record.TaskFinished) != 1 {
		t.Errorf("a run the supervisor called done did not finish: %v", kinds)
	}
}

// TestAVerdictOfAgainSendsItBackWithWhatIsMissing. A task sent round again
// without being told why is the same run a second time.
func TestAVerdictOfAgainSendsItBackWithWhatIsMissing(t *testing.T) {
	kinds, events := ranWith(t, "ACME-42", "verdict: again the delete path is still not idempotent", true)

	if count(kinds, record.TaskFinished) != 0 {
		t.Errorf("a run sent back also reported that it finished: %v", kinds)
	}

	if count(kinds, record.TaskRequeued) != 1 {
		t.Fatalf("the record holds no requeue: %v", kinds)
	}

	var told bool

	for _, e := range events {
		if e.Kind == record.TaskNoted && strings.Contains(e.Text, "delete path") {
			told = true
		}

		if e.Kind == record.TaskRequeued && e.Data["by"] != "supervisor" {
			t.Errorf("the requeue is filed under %q, want the supervisor", e.Data["by"])
		}
	}

	if !told {
		t.Error("the next run is not told what was missing")
	}
}

// TestAVerdictOfHumanStopsAndSaysWhatToDecide.
func TestAVerdictOfHumanStopsAndSaysWhatToDecide(t *testing.T) {
	kinds, events := ranWith(t, "ACME-43", "verdict: human the task does not say which of the two tables is authoritative", true)

	if count(kinds, record.TaskFinished) != 0 {
		t.Errorf("a run handed to a person also reported that it finished: %v", kinds)
	}

	last := events[len(events)-1]
	if last.Kind != record.TaskStuck {
		t.Fatalf("the record ends in %q, want task.stuck: %v", last.Kind, kinds)
	}

	if !strings.Contains(last.Text, "authoritative") {
		t.Errorf("the task is stuck without saying what a person has to decide:\n%s", last.Text)
	}
}

// TestAFlowThatDidNotAskIsNotValidated. It costs a model call on every task
// that finishes, so a flow that wants the last word asks for it.
func TestAFlowThatDidNotAskIsNotValidated(t *testing.T) {
	kinds, _ := ranWith(t, "ACME-44", "verdict: human this should never be read", false)

	if count(kinds, record.TaskFinished) != 1 {
		t.Errorf("a flow that asked for no validation was validated anyway: %v", kinds)
	}
}

// TestAnAnswerNobodyCanReadIsDone. A run that finished with its gates green
// is finished, and a judge that was unreachable is not the work's fault.
func TestAnAnswerNobodyCanReadIsDone(t *testing.T) {
	for _, answer := range []string{"", "I think it looks fine?", "the verdict is that it is done"} {
		if got, _ := verdictOf(answer); got != verdictDone {
			t.Errorf("%q reads as verdict %d, want done", answer, got)
		}
	}

	if got, why := verdictOf("VERDICT: AGAIN the tests are missing"); got != verdictAgain || why != "the tests are missing" {
		t.Errorf("an upper-case verdict reads as %d, %q", got, why)
	}
}
