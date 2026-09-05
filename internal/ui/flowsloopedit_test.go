package ui

// Building a loop in the designer, rather than by writing JSON.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
)

func building(t *testing.T) Model {
	t.Helper()

	m, _ := testModel(t, 100, 40)
	m = m.openFlows()
	m.flows.creating = true
	m.flows.flowName = "mine"
	m.flows.ensurePhase()

	return m
}

// TestAPhaseBecomesALoopAndBackAgain.
//
// The phase somebody has already written is what goes round: they typed an
// engine and a prompt for it, and turning the block on should keep both
// rather than making them start over.
func TestAPhaseBecomesALoopAndBackAgain(t *testing.T) {
	m := building(t)
	m.flows.phases[0].Name = "fix"
	m.flows.phases[0].Prompt = "read what failed"

	m = m.toggleLoop()

	loop := m.flows.phases[0].Loop
	if loop == nil {
		t.Fatal("the phase did not become a loop")
	}

	if len(loop.Phases) != 1 || loop.Phases[0].Prompt != "read what failed" {
		t.Errorf("what was already written was lost: %+v", loop.Phases)
	}

	// And back: the phase that went round is the phase again.
	m = m.toggleLoop()
	if m.flows.phases[0].Loop != nil || m.flows.phases[0].Prompt != "read what failed" {
		t.Errorf("turning the loop off lost the phase: %+v", m.flows.phases[0])
	}
}

// TestTheEnginesFieldsFollowThePhaseThatGoesRound. A loop has no engine of
// its own — what runs is inside it — so while the block is on, those fields
// are about the phase that repeats.
func TestTheEnginesFieldsFollowThePhaseThatGoesRound(t *testing.T) {
	m := building(t)
	m = m.toggleLoop()

	m.flows.edited().Prompt = "typed while looping"

	if got := m.flows.phases[0].Loop.Phases[0].Prompt; got != "typed while looping" {
		t.Errorf("the prompt went to %q instead of the phase inside the loop", got)
	}
}

// TestTheChecksAreWrittenOnePerLine, which is what a text field can hold and
// a reader can read: a name, a colon, and the command that says yes or no.
func TestTheChecksAreWrittenOnePerLine(t *testing.T) {
	m := building(t)
	m = m.toggleLoop()
	m = m.setLoopChecks("tests: go test ./...\ncoverage: make cover")

	until := m.flows.phases[0].Loop.Until
	if len(until) != 2 {
		t.Fatalf("two lines became %d checks: %+v", len(until), until)
	}

	if until[0].Name != "tests" || until[0].Command != "go test ./..." {
		t.Errorf("the first check is %+v", until[0])
	}

	if got := m.flows.loopChecksText(); !strings.Contains(got, "coverage: make cover") {
		t.Errorf("the field does not read back what was typed: %q", got)
	}
}

// TestALoopWithNothingToStopItIsRefused. It is the whole difference between
// a loop and a machine for spending a quota window on a wall.
func TestALoopWithNothingToStopItIsRefused(t *testing.T) {
	m := building(t)
	m.opts.Flows = flowsTestDir(t.TempDir())
	m = m.toggleLoop()
	m = m.setLoopChecks("")

	next, _ := m.saveCustomFlow()
	if _, err := flow.Resolve(next.opts.Flows, "mine"); err == nil {
		t.Error("a loop with no check was saved")
	}
}

// TestTheTurnsAreCappedAtSomethingReal. Zero turns is a loop that never runs
// and no cap is a loop that never stops.
func TestTheTurnsAreCappedAtSomethingReal(t *testing.T) {
	m := building(t)
	m = m.toggleLoop()

	if got := m.flows.phases[0].Loop.Max; got < 1 {
		t.Errorf("a loop was born with %d turns", got)
	}

	m = m.setLoopTurns(0)
	if got := m.flows.phases[0].Loop.Max; got < 1 {
		t.Errorf("the turns were set to %d", got)
	}
}

// TestThePurposeHoldsAParagraph. It is the field the list shows and the one
// a reader writes to remember why the flow exists, and one line of it was
// running out of row before the sentence was finished.
func TestThePurposeHoldsAParagraph(t *testing.T) {
	m := building(t)
	m.flows.field = flowFieldDescription

	m.flows.write("first line")

	if !m.flows.multiline() {
		t.Fatal("the purpose refuses a new line")
	}

	m.flows.write("\nsecond line")

	if m.flows.description != "first line\nsecond line" {
		t.Errorf("the purpose holds %q", m.flows.description)
	}

	// And the list, which has one row for it, shows it as one line.
	if got := flatten(m.flows.description); got != "first line second line" {
		t.Errorf("the list would draw %q", got)
	}

	// A name is not a paragraph: it is the file the flow is written to.
	m.flows.field = flowFieldName
	if m.flows.multiline() {
		t.Error("a newline is allowed in the flow's name")
	}
}
