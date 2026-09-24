package task

import (
	"context"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// panickingEngine dies in the middle of its phase, the way a bug in an
// engine adapter or a gate would take a run down.
type panickingEngine struct{}

func (panickingEngine) Name() string { return "panicking" }
func (panickingEngine) Run(context.Context, engine.Request) (engine.Result, error) {
	panic("the adapter fell over")
}

func (panickingEngine) Models() []engine.Choice                             { return nil }
func (panickingEngine) Efforts() []engine.Choice                            { return nil }
func (panickingEngine) CanThink() bool                                      { return false }
func (panickingEngine) Transcript(string, time.Time) ([]engine.Turn, error) { return nil, nil }
func (panickingEngine) RanOut(engine.Result, error) bool                    { return false }
func (panickingEngine) Locate() (string, error)                             { return "panicking", nil }
func (panickingEngine) CanResume() bool                                     { return false }

// TestARunThatPanicsClosesItsRecord. The run's claim came off on the way
// out, the log still ended at phase.started, and Reconcile skips a task
// nothing claims: the task read as running for good.
func TestARunThatPanicsClosesItsRecord(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "RUN-PANIC-1", "a run that dies without a word", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := flow.Flow{Name: "one", Phases: []flow.Phase{{Name: "implement", Engine: "panicking"}}}

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("the engine's panic did not reach the caller")
			}
		}()

		engines := map[string]engine.Engine{"panicking": panickingEngine{}}
		_ = Run(context.Background(), s, tk, f, engines, nil) //nolint:errcheck // it panics
	}()

	got, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if inFlight(got) {
		t.Fatalf("the log is still open after the run died; kinds are %v", kindsOf(got))
	}

	if last := got[len(got)-1]; last.Kind != record.TaskFailed || last.Text != errNoEnd.Error() {
		t.Errorf("last event = %s %q, want task.failed with errNoEnd", last.Kind, last.Text)
	}

	if _, alive, err := Alive(s, tk); err != nil || alive {
		t.Errorf("Alive = (%v, %v), want the marker released", alive, err)
	}
}
