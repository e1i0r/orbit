package flow

import "testing"

func TestAFlowThatSaysNothingGetsTheDefaultCap(t *testing.T) {
	f := Flow{Name: "task", Phases: []Phase{{Name: "implement", Engine: "claude"}}}
	if got := f.AttemptCap(); got != DefaultAttempts {
		t.Errorf("AttemptCap() = %d, want the default %d", got, DefaultAttempts)
	}
}

func TestAFlowMaySayHowManyAttemptsItAllows(t *testing.T) {
	f := Flow{Name: "task", Attempts: 5, Phases: []Phase{{Name: "implement", Engine: "claude"}}}
	if got := f.AttemptCap(); got != 5 {
		t.Errorf("AttemptCap() = %d, want 5", got)
	}
}

func TestAFlowThatAsksForNoAttemptAtAllIsRefused(t *testing.T) {
	f := Flow{Name: "task", Attempts: -1, Phases: []Phase{{Name: "implement", Engine: "claude"}}}
	if err := f.Validate(); err == nil {
		t.Error("Validate() = nil, want an error for a negative number of attempts")
	}
}
