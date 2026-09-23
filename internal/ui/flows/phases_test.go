package flows

// Adding a phase, taking one out, and walking between them.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/flow"
)

// TestWalkingThePhasesGoesRoundBothWays, and stops at neither end: the dial
// is a ring, so the phase after the last is the first.
func TestWalkingThePhasesGoesRoundBothWays(t *testing.T) {
	s, e := designing(t, 110, 45)
	s.field = flowFieldPhaseSelect

	n := len(s.phases)
	if n != 3 {
		t.Fatalf("this test needs three phases and the flow has %d", n)
	}

	at := s

	for i := range n * 2 {
		next, _ := at.handleFlowFieldDelta(1, e)
		if want := (i + 1) % n; next.activePhase != want {
			t.Fatalf("a step forward from %d left the designer on %d, want %d",
				at.activePhase, next.activePhase, want)
		}

		at = next
	}

	back := s

	for i := range n * 2 {
		next, _ := back.handleFlowFieldDelta(-1, e)
		if want := ((-(i + 1))%n + n) % n; next.activePhase != want {
			t.Fatalf("a step back from %d left the designer on %d, want %d",
				back.activePhase, next.activePhase, want)
		}

		back = next
	}
}

// TestWalkingThePhasesOfAFlowWithNone. A flow is written before its first
// phase is added, and the ring is a division by however many there are.
func TestWalkingThePhasesOfAFlowWithNone(t *testing.T) {
	s, e := designing(t, 110, 45)
	s.field = flowFieldPhaseSelect
	s.phases = nil
	s.activePhase = 0

	for _, d := range []int{1, -1} {
		next, _ := s.handleFlowFieldDelta(d, e)
		if next.activePhase != 0 {
			t.Errorf("a step of %+d with no phases left the designer on %d", d, next.activePhase)
		}
	}
}

// TestAPhaseAddedIsTheOneBeingEdited, and it is named after its place in
// the pipeline.
func TestAPhaseAddedIsTheOneBeingEdited(t *testing.T) {
	s, e := designing(t, 110, 45)

	was := len(s.phases)

	s.field = flowFieldAddPhase

	next, out := s.handleFlowFieldAction(e)
	if len(next.phases) != was+1 {
		t.Fatalf("adding a phase left %d of them, want %d", len(next.phases), was+1)
	}

	if next.activePhase != len(next.phases)-1 {
		t.Errorf("the phase added is %d and the designer is editing %d",
			len(next.phases)-1, next.activePhase)
	}

	if got := next.phases[len(next.phases)-1].Name; !strings.Contains(got, "4") {
		t.Errorf("the fourth phase is called %q", got)
	}

	if !strings.Contains(out.Said, "4") {
		t.Errorf("adding the fourth phase said %q", out.Said)
	}

	// And the cursor lands on its name, which is the first thing anybody
	// changes about a phase they just made.
	if next.field != flowFieldPhaseName {
		t.Errorf("adding a phase left the cursor on field %d", next.field)
	}
}

// TestDeletingTheLastPhaseLeavesTheCursorInside. The cursor is an index,
// and one that outlives its list points at a phase nobody can draw.
func TestDeletingTheLastPhaseLeavesTheCursorInside(t *testing.T) {
	s, e := designing(t, 110, 45)
	s.activePhase = len(s.phases) - 1
	s.field = flowFieldDelPhase

	next, out := s.handleFlowFieldAction(e)
	if len(next.phases) != len(s.phases)-1 {
		t.Fatalf("deleting left %d phases, want %d", len(next.phases), len(s.phases)-1)
	}

	if next.activePhase != len(next.phases)-1 {
		t.Errorf("after deleting the last phase the designer is editing %d of %d",
			next.activePhase, len(next.phases))
	}

	if out.Said == "" {
		t.Error("deleting a phase said nothing")
	}

	// The one left is drawn, and it is the one the cursor names.
	drawn := ansi.Strip(strings.Join(next.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n"))
	if !strings.Contains(drawn, next.phases[next.activePhase].Name) {
		t.Errorf("the phase being edited is not on the screen:\n%s", drawn)
	}
}

// TestAFlowKeepsAtLeastOnePhase. A flow of none is not a flow, and the
// designer says so rather than leaving an empty pipeline.
func TestAFlowKeepsAtLeastOnePhase(t *testing.T) {
	s, e := designing(t, 110, 45)
	s.phases = []flow.Phase{{Name: "only", Engine: "zeta"}}
	s.activePhase = 0
	s.field = flowFieldDelPhase

	next, out := s.handleFlowFieldAction(e)
	if len(next.phases) != 1 {
		t.Errorf("deleting the only phase left %d", len(next.phases))
	}

	if out.Said == "" {
		t.Error("refusing to delete the only phase said nothing")
	}
}

// TestDeletingAPhaseInTheMiddleKeepsTheRest, in order.
func TestDeletingAPhaseInTheMiddleKeepsTheRest(t *testing.T) {
	s, e := designing(t, 110, 45)
	s.activePhase = 1
	s.field = flowFieldDelPhase

	was := make([]string, 0, len(s.phases))
	for _, ph := range s.phases {
		was = append(was, ph.Name)
	}

	next, _ := s.handleFlowFieldAction(e)

	var left []string
	for _, ph := range next.phases {
		left = append(left, ph.Name)
	}

	if len(left) != len(was)-1 {
		t.Fatalf("deleting one of %d left %d", len(was), len(left))
	}

	for i, name := range []string{was[0], was[2]} {
		if left[i] != name {
			t.Errorf("phase %d is %q, want %q — the rest kept their order", i, left[i], name)
		}
	}
}
