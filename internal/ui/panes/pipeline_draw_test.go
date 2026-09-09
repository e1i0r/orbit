package panes

// The flow tree as it is drawn: a node per phase, what happened to each one,
// and what hangs off a node the reader opened.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
)

// tree is the world with a real flow resolved into it, opened.
func tree(t *testing.T, entries []view.Entry) Env {
	t.Helper()

	f, err := flow.Resolve(nil, flow.Default)
	if err != nil {
		t.Fatalf("this build cannot resolve its own default flow: %v", err)
	}

	e := world(t, entries)
	e.Flow = f
	e.RowOpen = func(int) bool { return true }

	return e
}

// TestEveryPhaseOfTheFlowIsANodeOfTheTree, whatever the record says about
// it: a flow is what the task will do, and a phase nothing has been written
// about yet is pending rather than absent.
func TestEveryPhaseOfTheFlowIsANodeOfTheTree(t *testing.T) {
	e := tree(t, nil)

	rows, heads := Pipeline(e)
	if len(heads) != len(e.Flow.Phases) {
		t.Errorf("the tree offers %d nodes for a flow of %d phases", len(heads), len(e.Flow.Phases))
	}

	got := text(rows)
	for _, phase := range e.Flow.Phases {
		if !strings.Contains(got, phase.Name) {
			t.Errorf("the tree does not name the phase %q:\n%s", phase.Name, got)
		}
	}

	if !strings.Contains(got, "pending") {
		t.Errorf("a flow nothing has run reads as something other than pending:\n%s", got)
	}
}

// TestANodeSaysWhereItsPhaseGotTo, and an opened one what it was set up with
// and what it wrote.
func TestANodeSaysWhereItsPhaseGotTo(t *testing.T) {
	e := tree(t, nil)

	name := e.Flow.Phases[0].Name
	e.Entries = []view.Entry{
		{Kind: "phase.started", Phase: name, At: ago(30 * time.Minute), Engine: "claude", Model: "opus"},
		{
			Kind: "phase.finished", Phase: name, At: ago(20 * time.Minute), Cost: 0.5,
			Text: "wrote the endpoint and its test",
		},
	}

	got := text(first(Pipeline(e)))
	for _, want := range []string{"completed", "claude", "opus", "wrote the endpoint", "$0.5000"} {
		if !strings.Contains(got, want) {
			t.Errorf("an opened node does not carry %q:\n%s", want, got)
		}
	}
}

// TestAPhaseThatBrokeCarriesTheReasonOrItsExitCode. A node that says only
// "failed" is a node a reader has to leave the tree to understand.
func TestAPhaseThatBrokeCarriesTheReasonOrItsExitCode(t *testing.T) {
	e := tree(t, nil)
	name := e.Flow.Phases[0].Name

	e.Entries = []view.Entry{
		{Kind: "phase.started", Phase: name, At: ago(30 * time.Minute)},
		{Kind: "phase.failed", Phase: name, At: ago(20 * time.Minute), Cause: "the build would not link"},
	}

	if got := text(first(Pipeline(e))); !strings.Contains(got, "the build would not link") {
		t.Errorf("a broken phase does not say why:\n%s", got)
	}

	// And with no reason written down, the exit code is what there is.
	e.Entries = []view.Entry{
		{Kind: "phase.started", Phase: name, At: ago(30 * time.Minute)},
		{Kind: "phase.failed", Phase: name, At: ago(20 * time.Minute), Exit: "137"},
	}

	if got := text(first(Pipeline(e))); !strings.Contains(got, "137") {
		t.Errorf("a broken phase with no reason does not name its exit code:\n%s", got)
	}
}

// TestAPhaseWaitingAtItsGateSaysSo, which is the one standing that asks
// something of the reader.
func TestAPhaseWaitingAtItsGateSaysSo(t *testing.T) {
	e := tree(t, nil)
	name := e.Flow.Phases[0].Name

	e.Entries = []view.Entry{{Kind: "phase.waiting", Phase: name, At: ago(time.Minute), Cause: "gate: tests"}}

	if got := text(first(Pipeline(e))); !strings.Contains(got, "waiting at gate") {
		t.Errorf("a phase stopped at its gate does not say so:\n%s", got)
	}
}

// TestThePhaseInFlightIsMarkedAsGoing. The task says which phase it is in,
// and the tree is where a reader watches it move.
func TestThePhaseInFlightIsMarkedAsGoing(t *testing.T) {
	e := tree(t, nil)
	e.Task.Band, e.Task.Phase = view.Running, e.Flow.Phases[0].Name

	if got := text(first(Pipeline(e))); !strings.Contains(got, "in progress") {
		t.Errorf("the phase the run is in is not marked as going:\n%s", got)
	}
}

// TestTheVerbsAskedForByHandHangOffTheSameTrunk. They are not phases and
// nothing in a flow put them there, so they are under a heading of their
// own — but they happened to this task, in this order, which is the question
// the tree answers.
func TestTheVerbsAskedForByHandHangOffTheSameTrunk(t *testing.T) {
	e := tree(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "PR", By: "supervisor", At: ago(10 * time.Minute)},
		{Kind: "deliver.answered", Verb: "PR", Text: "opened #12", At: ago(9 * time.Minute)},
		{Kind: "deliver.asked", Verb: "MERGE", By: "supervisor", At: ago(8 * time.Minute)},
		{Kind: "deliver.answered", Verb: "MERGE", Cause: "checks are still red", At: ago(7 * time.Minute)},
		{Kind: "deliver.asked", Verb: "FIX CHECKS", By: "supervisor", At: ago(time.Minute)},
	})

	got := text(first(Pipeline(e)))
	for _, want := range []string{
		"Asked for by hand",
		"opened #12",                 // one that came back
		"came back broken",           // one that broke
		"checks are still red",       // and why
		"handed over, still working", // one that has not
		"supervisor",                 // what was handed the work
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the tree does not carry %q under the verbs:\n%s", want, got)
		}
	}
}

// TestATaskThatLeftTheBoardDrawsNoTree.
func TestATaskThatLeftTheBoardDrawsNoTree(t *testing.T) {
	e := tree(t, nil)
	e.Gone = true

	if got := text(first(Pipeline(e))); !strings.Contains(got, "no longer on the board") {
		t.Errorf("the tree of a task that left drew %q", got)
	}
}
