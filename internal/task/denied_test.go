package task

// A phase that was refused what it needed, and left nothing behind.

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// refusedEngine answers the way one does when a headless run is denied a
// tool: it says it could not, and exits zero.
func refusedEngine(said, tool string) *engine.Fake {
	f := engine.NewFake(said)
	f.Events = []engine.StreamEvent{
		{Type: "refusal", Refusal: engine.StreamRefusal{Tool: tool, Input: "CONTRIBUTING.md"}},
	}

	return f
}

// TestAPhaseRefusedAndEmptyIsNotFinished is the whole of it. Read by its exit
// code alone this was a success, which is how a task that did nothing came to
// sit in done — with a "finished" on somebody's phone.
func TestAPhaseRefusedAndEmptyIsNotFinished(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DENY-1", "write CONTRIBUTING.md", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := refusedEngine("I could not write it: a permission prompt was not approved", "Write")

	runErr := Run(context.Background(), s, tk, oneEngineFlow("fake"), fakes(fake), nil)
	if runErr == nil {
		t.Fatal("Run: a phase that was denied and wrote nothing was called a success")
	}

	if !strings.Contains(runErr.Error(), "what the phase is allowed") {
		t.Errorf("the refusal does not say where to look: %v", runErr)
	}

	kinds := kindsFor(t, s, tk)
	if count(kinds, record.PhaseDenied) != 1 {
		t.Errorf("no phase was written down as denied:\n%v", kinds)
	}

	if count(kinds, record.PhaseFinished) != 0 || count(kinds, record.TaskFinished) != 0 {
		t.Errorf("it was written down as finished:\n%v", kinds)
	}

	// Which tool, so the reader knows what the posture is missing.
	if got := lastOfKind(t, s, tk, record.PhaseDenied); got.Data["tool"] != "Write" {
		t.Errorf("it says it was denied %q, want Write", got.Data["tool"])
	}
}

// TestAPhaseThatWasRefusedAndCarriedOnIsFine. A refusal on its own is
// ordinary: a phase turned down once that went another way did the work, and
// the work is on disk to prove it.
func TestAPhaseThatWasRefusedAndCarriedOnIsFine(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DENY-2", "write something", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := refusedEngine("the first way was refused, so I did it another way", "WebFetch")

	// The stand-in writes nothing itself, so a gate leaves what a phase that
	// did the work would have left. Gates run in the worktree, which is the
	// same place the check below looks.
	wrote := flow.Flow{Name: "task", Phases: []flow.Phase{{
		Name: "implement", Engine: "fake",
		Gates: []flow.Gate{{Name: "write", Command: "echo hi > NOTES.md"}},
	}}}

	if err := Run(context.Background(), s, tk, wrote, fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	kinds := kindsFor(t, s, tk)
	if count(kinds, record.PhaseDenied) != 0 {
		t.Errorf("a phase that did the work was written down as denied:\n%v", kinds)
	}

	if count(kinds, record.TaskFinished) != 1 {
		t.Errorf("it did not finish:\n%v", kinds)
	}
}
