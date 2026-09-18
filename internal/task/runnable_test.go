package task

// Whether a flow can be walked at all, asked before anything is spent.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
)

// looping is a loop of one phase, with something that says when it is done.
func looping(on string) *flow.Loop {
	return &flow.Loop{
		Phases: []flow.Phase{{Name: "fix", Engine: on}},
		Until:  []flow.Gate{{Name: "tests", Command: "go test ./..."}},
		Max:    2,
	}
}

// TestAFlowIsRefusedForAnyPhaseThatCannotRun.
//
// Before the first engine is asked for anything, because the alternative is
// a run that does three phases' work and stops at the fourth over a name
// nobody had to wait to find out was wrong.
//
// A loop names no engine and its phases do, so the phases inside one are
// asked the same question — and a loop that is fine is not the end of the
// list: the phases after it have to be asked too.
func TestAFlowIsRefusedForAnyPhaseThatCannotRun(t *testing.T) {
	have := map[string]engine.Engine{"fake": engine.NewFake("done")}

	fine := flow.Flow{Name: "task", Phases: []flow.Phase{
		{Name: "implement", Engine: "fake"},
		{Name: "check", Loop: looping("fake")},
		{Name: "review", Engine: "fake"},
	}}

	if err := runnable(fine, have); err != nil {
		t.Fatalf("a flow every phase of which can run was refused: %v", err)
	}

	for _, one := range []struct {
		why  string
		flow flow.Flow
	}{
		{
			"a phase of its own wants an engine nobody configured",
			flow.Flow{Name: "task", Phases: []flow.Phase{
				{Name: "implement", Engine: "fake"},
				{Name: "review", Engine: "codex"},
			}},
		},
		{
			"a phase inside a loop wants one",
			flow.Flow{Name: "task", Phases: []flow.Phase{
				{Name: "check", Loop: looping("codex")},
			}},
		},
		{
			"and a loop that is fine is not the end of the list",
			flow.Flow{Name: "task", Phases: []flow.Phase{
				{Name: "check", Loop: looping("fake")},
				{Name: "review", Engine: "codex"},
			}},
		},
	} {
		err := runnable(one.flow, have)
		if err == nil {
			t.Errorf("a flow was accepted where %s", one.why)
			continue
		}

		if !strings.Contains(err.Error(), "codex") {
			t.Errorf("the refusal is %q, and does not name the engine that is missing", err)
		}
	}
}
