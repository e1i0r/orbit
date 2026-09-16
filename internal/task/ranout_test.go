package task

// A run whose engine had nothing left to spend.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// spentEngine is one that stops the way a provider stops it: a non-zero exit
// with the refusal above it.
type spentEngine struct{ said string }

func (spentEngine) Name() string                                        { return "fake" }
func (spentEngine) CanResume() bool                                     { return false }
func (spentEngine) Models() []engine.Choice                             { return nil }
func (spentEngine) Efforts() []engine.Choice                            { return nil }
func (spentEngine) CanThink() bool                                      { return false }
func (spentEngine) Transcript(string, time.Time) ([]engine.Turn, error) { return nil, nil }
func (spentEngine) RanOut(out engine.Result, err error) bool {
	return engine.NewClaude().RanOut(out, err)
}
func (spentEngine) Locate() (string, error) { return "spent", nil }

func (e spentEngine) Run(context.Context, engine.Request) (engine.Result, error) {
	return engine.Result{Output: e.said}, errRanOut
}

// errRanOut is what exec gives a caller for a program that exited non-zero,
// which is the only vocabulary it has: the engine's own words are above it,
// in what it printed.
var errRanOut = errors.New("exit status 1")

// ranOutRun is a run over a task written for this, with an engine that stops
// the way a provider stops one.
func ranOutRun(t *testing.T, said string) (*store.Store, Task, phaseRun) {
	t.Helper()

	s, tk := nowhere(t, "ACME-1", "do the thing")

	return s, tk, phaseRun{
		store: s, task: tk,
		eng:   spentEngine{said: said},
		phase: flow.Phase{Name: "implement"},
	}
}

// lastPhaseEnd is the kind of the last event that ended a phase.
func lastPhaseEnd(t *testing.T, s *store.Store, tk Task) string {
	t.Helper()

	for _, e := range eventsOf(t, s, tk) {
		switch e.Kind {
		case record.PhaseRanOut, record.PhaseFailed:
			return e.Kind
		}
	}

	return ""
}

// TestAnEngineWithNothingLeftIsNotAnEngineThatBroke.
//
// The whole of why this ending exists: a reader who comes back to a task
// that says only "it broke" has to open the log to find out which of the two
// it was, and the two send them to do opposite things.
func TestAnEngineWithNothingLeftIsNotAnEngineThatBroke(t *testing.T) {
	for _, c := range []struct {
		name string
		said string
		want string
	}{
		{"the allowance is gone", "Claude AI usage limit reached", record.PhaseRanOut},
		{"the engine broke", "./main.go:12:2: declared and not used", record.PhaseFailed},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, tk, r := ranOutRun(t, c.said)
			_ = r.broke(context.Background(), engine.Result{Output: c.said}, errRanOut) //nolint:errcheck // the end is what is read

			if kind := lastPhaseEnd(t, s, tk); kind != c.want {
				t.Errorf("the run ended as %q, want %q", kind, c.want)
			}
		})
	}
}
