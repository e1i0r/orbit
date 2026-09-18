package task

// Which phase a run says it is in, told to the gate and written into the
// record.
//
// The number is how every reader lines a run up against its flow: the bar
// draws "2/4", the gate is asked about phase 2, and the events of phase 2
// carry it. Counted from anywhere else, all three agree with each other and
// with nothing a person can see.

import (
	"context"
	"strconv"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// askedGate remembers which phase it was asked about, in order.
type askedGate struct{ asked []int }

func (g *askedGate) Before(_ context.Context, _ Task, _ flow.Phase, n int) (Go, error) {
	g.asked = append(g.asked, n)

	return Continue, nil
}

// TestTheGateIsToldWhichPhaseOfTheFlowItIs.
func TestTheGateIsToldWhichPhaseOfTheFlowItIs(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-37", "three phases", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	three := flow.Flow{Name: "task", Phases: []flow.Phase{
		{Name: "plan", Engine: "fake"},
		{Name: "implement", Engine: "fake"},
		{Name: "review", Engine: "fake"},
	}}

	gate := &askedGate{}
	if err := Run(context.Background(), s, tk, three, fakes(engine.NewFake("done")), gate); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []int{1, 2, 3}
	if len(gate.asked) != len(want) {
		t.Fatalf("the gate was asked about %v, want %v", gate.asked, want)
	}

	for i, n := range want {
		if gate.asked[i] != n {
			t.Errorf("the gate was asked about phase %d where the run is at %d", gate.asked[i], n)
		}
	}
}

// TestEachPhaseOfALoopSaysWhichOfThemItIs.
//
// A loop's phases are numbered within the turn, because that is what the
// record has to be able to say: "the second phase of the loop wrote this".
// Numbered from zero, every turn's first phase would be a phase 0 that
// appears nowhere in the flow a reader is looking at.
func TestEachPhaseOfALoopSaysWhichOfThemItIs(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-38", "a loop of two", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	twice := flow.Flow{Name: "tdd", Phases: []flow.Phase{
		{Name: "green", Loop: &flow.Loop{
			Phases: []flow.Phase{{Name: "fix", Engine: "fake"}, {Name: "recheck", Engine: "fake"}},
			Until:  []flow.Gate{{Name: "unit", Command: "true"}},
			Max:    2,
		}},
	}}

	if err := Run(context.Background(), s, tk, twice, fakes(engine.NewFake("done")), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var numbered []string

	for _, e := range mustEvents(t, s, tk) {
		if e.Kind == record.PhaseStarted {
			numbered = append(numbered, e.Phase+"="+e.Data["n"])
		}
	}

	if len(numbered) != 2 {
		t.Fatalf("the record holds %d phases that started: %v", len(numbered), numbered)
	}

	for i, want := range []string{"fix=" + strconv.Itoa(1), "recheck=" + strconv.Itoa(2)} {
		if numbered[i] != want {
			t.Errorf("phase %d of the turn is written down as %q, want %q", i+1, numbered[i], want)
		}
	}
}
