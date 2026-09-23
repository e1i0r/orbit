package ui

// The flow row: every flow visible, and a click answered where it was drawn.

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/prose"
)

// startedOn is the start dialog, open on one task.
func startedOn(t *testing.T, id string) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)

	next, _ := onto(t, m, id).openStart()

	open := asModel(t, next)
	if open.screen != screenStart {
		t.Fatalf("openStart left screen %v, want screenStart", open.screen)
	}

	return open
}

// TestTheStartDialogDrawsEveryFlow.
//
// It drew the one it was on and, off to the right, the order `f` would
// visit the others in — which at a hundred columns was cut off. Choosing
// `gated` out of eight meant pressing f until it appeared, with nothing
// saying how many were left or that you had just gone past it.
func TestTheStartDialogDrawsEveryFlow(t *testing.T) {
	m := startedOn(t, "ACME-2698")

	drawn := ansi.Strip(strings.Join(m.flowBlock(100), "\n"))

	for _, f := range m.start.flows {
		if !strings.Contains(drawn, f.name) {
			t.Errorf("the row does not show %q, so nobody can pick it:\n%s", f.name, drawn)
		}
	}

	// And the one in use is marked, or a row of eight says nothing about
	// which is about to run.
	if !strings.Contains(drawn, "●") {
		t.Errorf("no flow is marked as the chosen one:\n%s", drawn)
	}
}

// TestAClickPicksTheFlowItLandedOn walks the row cell by cell and checks
// that what answers a column is what was drawn in it.
//
// The two halves are worked out in different files from the same widths,
// which is the shape every click bug in this window has had.
func TestAClickPicksTheFlowItLandedOn(t *testing.T) {
	m := startedOn(t, "ACME-2698")
	if len(m.start.flows) < 2 {
		t.Skip("this fixture has one flow, so there is nothing to pick between")
	}

	_, placed := m.flowRow(m.frame.Body.W)
	if len(placed) != len(m.start.flows) {
		t.Fatalf("the row placed %d pills for %d flows", len(placed), len(m.start.flows))
	}

	// Every pill answers its own index, and no two overlap — on its own
	// line, because the row wraps and the same column two lines down is a
	// different flow.
	for want, p := range placed {
		for x := p.At; x < p.At+p.Cells; x++ {
			got, on := prose.At(placed, x, p.Line)
			if !on || got != want {
				t.Fatalf("column %d of line %d is inside pill %d and answers (%d, %v)",
					x, p.Line, want, got, on)
			}
		}
	}

	// And the click reaches the model: pick the last one.
	last := len(m.start.flows) - 1

	next, _ := m.pickFlow(last)
	if asModel(t, next).start.at != last {
		t.Errorf("a click on flow %d left the dialog on %d", last, asModel(t, next).start.at)
	}

	// Out of range changes nothing rather than clamping.
	same, _ := m.pickFlow(len(m.start.flows) + 3)
	if asModel(t, same).start.at != m.start.at {
		t.Error("a click outside the row moved the choice")
	}
}

// TestTheKeyStillCycles: anybody who learned f does not have to unlearn it.
func TestTheKeyStillCycles(t *testing.T) {
	m := startedOn(t, "ACME-2698")
	if len(m.start.flows) < 2 {
		t.Skip("one flow, nothing to cycle")
	}

	was := m.start.at
	if m.cycleFlow().start.at == was {
		t.Error("f no longer moves to the next flow")
	}
}

// TestEveryDialogTargetIsRouted, the same guard the compose targets have.
//
// It found one on the way in: point.DialogPhase was produced by the hit
// test and named nowhere, so a click on a phase row in this dialog took
// the click and did nothing with it. The phase rows are a preview of what
// the flow will do rather than a thing to choose, so they answer nothing
// at all now.
func TestEveryDialogTargetIsRouted(t *testing.T) {
	routing, err := os.ReadFile("mouse.go")
	if err != nil {
		t.Fatalf("read the mouse routing: %v", err)
	}

	for _, kind := range []string{"DialogFlow", "DialogSwitch"} {
		if !strings.Contains(string(routing), "point."+kind) {
			t.Errorf("point.%s is a target the dialog answers and mouse.go never names it", kind)
		}
	}
}

// TestTheRowWrapsRatherThanCuttingFlowsOff.
//
// Eight flows do not fit on one line at a hundred columns, which is the
// width Elio works at. Truncating there would be the same failure this
// change is for, with a nicer first line: choices that cannot be seen.
func TestTheRowWrapsRatherThanCuttingFlowsOff(t *testing.T) {
	m := startedOn(t, "ACME-2698")

	for _, w := range []int{100, 120, 180} {
		rows := m.flowBlock(w)
		drawn := ansi.Strip(strings.Join(rows, "\n"))

		for _, f := range m.start.flows {
			if !strings.Contains(drawn, f.name) {
				t.Errorf("at %d columns %q is not drawn anywhere:\n%s", w, f.name, drawn)
			}
		}

		for i, r := range rows {
			if got := ansi.StringWidth(r); got > w {
				t.Errorf("at %d columns line %d is %d wide:\n%s", w, i, got, ansi.Strip(r))
			}
		}
	}
}

// TestThePlanMovesWithTheRow: the blocks under the flow row are placed
// from what it actually took, or a second line would sit on top of the
// engine dials.
func TestThePlanMovesWithTheRow(t *testing.T) {
	m := startedOn(t, "ACME-2698")

	narrow := m.startLayout(100)
	wide := m.startLayout(200)

	if narrow.nFlow != len(m.flowBlock(100)) {
		t.Errorf("the plan says %d flow lines and the draw makes %d", narrow.nFlow, len(m.flowBlock(100)))
	}

	if narrow.config <= narrow.flow+narrow.nFlow-1 {
		t.Errorf("the dials at line %d land inside a flow block of %d lines from %d",
			narrow.config, narrow.nFlow, narrow.flow)
	}

	if wide.nFlow > narrow.nFlow {
		t.Errorf("a wider terminal wrapped more: %d lines at 200, %d at 100", wide.nFlow, narrow.nFlow)
	}
}
