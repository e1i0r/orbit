package task

// A fact that stops the work is a gate.

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/knowledge"
)

func stopping(phrase, check string) knowledge.Fact {
	f := aFact(phrase, knowledge.Scope{Kind: knowledge.General})
	f.Stops, f.Check = true, check

	return f
}

// TestOnlyAFactThatCanCheckItselfBecomesAGate.
//
// This is where "stops" stops being a word in a prompt. A sentence in the
// context is advice the model may weigh against everything else it was told;
// a gate is a command that runs after it and sends the work back. The two
// are different powers and only one of them needs no trust.
func TestOnlyAFactThatCanCheckItselfBecomesAGate(t *testing.T) {
	gates := knowledgeGates([]knowledge.Fact{
		aFact("The PRs are written in English.", knowledge.Scope{Kind: knowledge.General}),
		stopping("No UPDATE or DELETE in ledger.", "! git diff | grep -q 'UPDATE ledger'"),
		func() knowledge.Fact {
			asked := aFact("Coverage stays above 90%.", knowledge.Scope{Kind: knowledge.General})
			asked.Stops = true // and no check: it cannot enforce itself

			return asked
		}(),
	})

	if len(gates) != 1 {
		t.Fatalf("%d facts became gates, want the one that brought a check: %+v", len(gates), gates)
	}

	if gates[0].Command != "! git diff | grep -q 'UPDATE ledger'" {
		t.Errorf("the gate runs %q, want the fact's own check", gates[0].Command)
	}
}

// TestTheGateIsNamedAfterWhatItIsAbout. The refusal a later attempt reads
// names the gate, so the name has to be the thing that was broken — "exit 1"
// against a check nobody can see is a wall with no sign on it.
func TestTheGateIsNamedAfterWhatItIsAbout(t *testing.T) {
	gates := knowledgeGates([]knowledge.Fact{
		stopping("No UPDATE or DELETE in ledger. Reconcile marks, it does not correct.", "false"),
	})

	if len(gates) != 1 {
		t.Fatalf("%d gates, want 1", len(gates))
	}

	if !strings.Contains(gates[0].Name, "No UPDATE or DELETE in ledger") {
		t.Errorf("the gate is called %q, which does not say what it is about", gates[0].Name)
	}
}

// TestAFactThatIsOffGatesNothing. Turning one off has to stop it refusing
// work, or it is not off.
func TestAFactThatIsOffGatesNothing(t *testing.T) {
	off := stopping("No UPDATE in ledger.", "false")
	off.Off = true

	if gates := knowledgeGates([]knowledge.Fact{off}); len(gates) != 0 {
		t.Errorf("a fact that was turned off still gates: %+v", gates)
	}
}

// TestThePhasesOwnGatesRunFirst. A flow's gates are about the work that
// phase was asked to do; the facts are standing rules that were true before
// it started. The specific failure is the more useful one to be told about.
func TestThePhasesOwnGatesRunFirst(t *testing.T) {
	p := flow.Phase{Name: "implement", Gates: []flow.Gate{{Name: "build", Command: "go build ./..."}}}

	all := gatesOf(p, []knowledge.Fact{stopping("No UPDATE in ledger.", "false")})
	if len(all) != 2 || all[0].Name != "build" {
		t.Errorf("the gates run as %+v, want the phase's own first", all)
	}
}

// TestAStandingRuleRefusesTheWork, end to end: a fact written to the store
// with a check that fails sends the phase back, and the refusal a later
// attempt reads names the sentence that was broken.
//
// The phase declares no gates of its own. Nothing about the flow, the task or
// the phase says this check exists — it is standing in the repository, and it
// runs because it is there.
func TestAStandingRuleRefusesTheWork(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "KNOW-GATE-1", "touch the ledger", "quick")
	if err != nil {
		t.Fatal(err)
	}

	saved := stopping("No UPDATE or DELETE in ledger. Reconcile marks, it does not correct.", "exit 3")
	saved.Scope = knowledge.Scope{Kind: knowledge.Repo, Repo: r.Path}

	if _, err := knowledge.NewStore(s.Root()).Save(saved); err != nil {
		t.Fatalf("save the fact: %v", err)
	}

	refused, err := runGates(context.Background(), s, tk, flow.Phase{Name: "implement"}, 1, t.TempDir(), engine.Result{})
	if err != nil {
		t.Fatalf("runGates: %v", err)
	}

	if refused == nil {
		t.Fatal("a standing rule with a failing check let the work through")
	}

	if !strings.Contains(refused.Gate, "No UPDATE or DELETE in ledger") {
		t.Errorf("the refusal names %q, which does not say what was broken", refused.Gate)
	}

	if refused.Exit != 3 {
		t.Errorf("the refusal carries exit %d, want the check's own 3", refused.Exit)
	}
}
