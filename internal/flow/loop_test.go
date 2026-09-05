package flow

import "testing"

func loopFlow(l *Loop) Flow {
	return Flow{Name: "tdd", Phases: []Phase{
		{Name: "implement", Engine: "claude"},
		{Name: "verify", Loop: l},
	}}
}

// TestALoopNeedsACheckACapAndSomethingToRepeat. Every one of the three is
// what keeps a loop from being a doom loop: without a check it never ends,
// without a cap an impossible check burns the whole quota window, and
// without phases there is nothing to go round.
func TestALoopNeedsACheckACapAndSomethingToRepeat(t *testing.T) {
	for name, l := range map[string]*Loop{
		"no check":  {Phases: []Phase{{Name: "fix", Engine: "claude"}}, Max: 3},
		"no cap":    {Phases: []Phase{{Name: "fix", Engine: "claude"}}, Until: []Gate{{Name: "unit", Command: "go test ./..."}}},
		"no phases": {Until: []Gate{{Name: "unit", Command: "go test ./..."}}, Max: 3},
		"a check with no command": {
			Phases: []Phase{{Name: "fix", Engine: "claude"}},
			Until:  []Gate{{Name: "unit"}}, Max: 3,
		},
		"an inner phase with no engine": {
			Phases: []Phase{{Name: "fix"}},
			Until:  []Gate{{Name: "unit", Command: "go test ./..."}}, Max: 3,
		},
	} {
		if err := loopFlow(l).Validate(); err == nil {
			t.Errorf("a loop with %s was accepted", name)
		}
	}
}

// TestALoopIsAcceptedWhenItHasAllThree.
func TestALoopIsAcceptedWhenItHasAllThree(t *testing.T) {
	err := loopFlow(&Loop{
		Phases: []Phase{{Name: "fix", Engine: "claude"}},
		Until:  []Gate{{Name: "unit", Command: "go test ./..."}},
		Max:    3,
	}).Validate()
	if err != nil {
		t.Errorf("a whole loop was refused: %v", err)
	}
}

// TestALoopInsideALoopIsRefused. The cap of the outer one would stop
// bounding anything, and a reader could not say how many times the innermost
// phase may run.
func TestALoopInsideALoopIsRefused(t *testing.T) {
	inner := &Loop{
		Phases: []Phase{{Name: "fix", Engine: "claude"}},
		Until:  []Gate{{Name: "unit", Command: "true"}}, Max: 2,
	}
	outer := &Loop{
		Phases: []Phase{{Name: "inner", Loop: inner}},
		Until:  []Gate{{Name: "all", Command: "true"}}, Max: 2,
	}

	if err := loopFlow(outer).Validate(); err == nil {
		t.Error("a loop inside a loop was accepted")
	}
}

// TestAPhaseIsAnEngineOrALoopAndNotBoth.
func TestAPhaseIsAnEngineOrALoopAndNotBoth(t *testing.T) {
	f := Flow{Name: "tdd", Phases: []Phase{{
		Name: "verify", Engine: "claude",
		Loop: &Loop{
			Phases: []Phase{{Name: "fix", Engine: "claude"}},
			Until:  []Gate{{Name: "unit", Command: "true"}}, Max: 2,
		},
	}}}

	if err := f.Validate(); err == nil {
		t.Error("a phase that is both an engine and a loop was accepted")
	}
}

// TestTheCoverageFlowIsALoopThatCanRun.
//
// The loop block shipped and nothing used it: no built-in declared one, so
// the whole feature was a function nobody could invoke. This is the worked
// example — and it has to be one that runs, not one that reads well.
func TestTheCoverageFlowIsALoopThatCanRun(t *testing.T) {
	f, err := Builtin("coverage")
	if err != nil {
		t.Fatalf("Builtin(coverage): %v", err)
	}

	var loops int

	for _, p := range f.Phases {
		if p.Loop == nil {
			continue
		}

		loops++

		if p.Loop.Max < 1 {
			t.Errorf("the loop goes round %d times", p.Loop.Max)
		}

		if len(p.Loop.Until) == 0 {
			t.Error("the loop has nothing that says it can stop")
		}

		if len(p.Loop.Phases) == 0 {
			t.Error("the loop repeats no phases")
		}

		// Fed the checks' output, or the next turn is the same turn again.
		for _, inner := range p.Loop.Phases {
			if !inner.FeedOutput {
				t.Errorf("the phase %q inside the loop is not told what failed", inner.Name)
			}
		}
	}

	if loops != 1 {
		t.Errorf("the coverage flow declares %d loops, want one", loops)
	}
}
