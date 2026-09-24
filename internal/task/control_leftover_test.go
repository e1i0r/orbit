package task

import (
	"context"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
)

// pressedDuringLast is a gate that lets every phase go and, at the last
// one, has a reader press a word: one written while the last phase runs,
// with no gate after it to take it.
type pressedDuringLast struct {
	Gate
	press func(Task) error
	last  string
}

func (g pressedDuringLast) Before(ctx context.Context, t Task, p flow.Phase, n int) (Go, error) {
	decided, err := g.Gate.Before(ctx, t, p, n)
	if err == nil && p.Name == g.last {
		err = g.press(t)
	}

	return decided, err
}

// TestAWordLeftByARunDoesNotMoveTheNext. A skip pressed during the last
// phase had no gate after it to take it, so the next run's first gate did,
// and dropped that run's first phase with nothing in the record to say so.
// The word goes with the run it was written for.
func TestAWordLeftByARunDoesNotMoveTheNext(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	f := twoFlow()

	press := pressedDuringLast{
		Gate:  FileGate(s, time.Second),
		press: func(t Task) error { return Control(s, t, wordSkip) },
		last:  f.Phases[len(f.Phases)-1].Name,
	}

	first := engine.NewFake("done")
	if err := Run(context.Background(), s, tk, f, fakes(first), press); err != nil {
		t.Fatalf("the first run: %v", err)
	}

	fake := engine.NewFake("done")

	if err := Run(context.Background(), s, tk, f, fakes(fake), press.Gate); err != nil {
		t.Fatalf("the second run: %v", err)
	}

	if len(fake.Calls) != len(f.Phases) {
		t.Errorf("the second run called the engine %d times, want %d: "+
			"a word left by the first run skipped a phase", len(fake.Calls), len(f.Phases))
	}
}

// TestAWordWrittenBeforeARunStillReachesIt. A pause asked for while nothing
// runs is for the run that comes next, and clearing words at the end of a
// run must not clear that one.
func TestAWordWrittenBeforeARunStillReachesIt(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)

	if err := Control(s, tk, wordSkip); err != nil {
		t.Fatalf("Control: %v", err)
	}

	fake := engine.NewFake("done")
	gate := FileGate(s, time.Second)

	if err := Run(context.Background(), s, tk, twoFlow(), fakes(fake), gate); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != len(twoFlow().Phases)-1 {
		t.Errorf("the engine was called %d times, want the first phase skipped", len(fake.Calls))
	}
}
