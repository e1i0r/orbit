package task

// A fact that stops the work is a gate.

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/record"
)

func stopping(phrase, check string) knowledge.Rule {
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
	gates := knowledgeGates([]knowledge.Rule{
		aFact("The PRs are written in English.", knowledge.Scope{Kind: knowledge.General}),
		stopping("No UPDATE or DELETE in ledger.", "! git diff | grep -q 'UPDATE ledger'"),
		func() knowledge.Rule {
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
	gates := knowledgeGates([]knowledge.Rule{
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
	off.State = knowledge.Off

	if gates := knowledgeGates([]knowledge.Rule{off}); len(gates) != 0 {
		t.Errorf("a fact that was turned off still gates: %+v", gates)
	}
}

// TestThePhasesOwnGatesRunFirst. A flow's gates are about the work that
// phase was asked to do; the facts are standing rules that were true before
// it started. The specific failure is the more useful one to be told about.
func TestThePhasesOwnGatesRunFirst(t *testing.T) {
	p := flow.Phase{Name: "implement", Gates: []flow.Gate{{Name: "build", Command: "go build ./..."}}}

	all := gatesOf(p, []knowledge.Rule{stopping("No UPDATE in ledger.", "false")})
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

// TestAGateIsNamedAfterWhatItMeans.
//
// A check like `! git diff | grep -q X` prints nothing at all when it fails,
// so the refusal a later attempt reads has only the name to go on: "gate `No
// UPDATE or DELETE in ledger` refused it" tells a model what it broke, where
// "gate `rule-7` refused it, exit 1" is a wall with no sign on it.
//
// Cut at the length it is cut at and not one character sooner: the sentence
// carries the meaning, and a rule of exactly that length is one somebody
// wrote to fit.
func TestAGateIsNamedAfterWhatItMeans(t *testing.T) {
	exact := strings.Repeat("a", gateName)
	if got := gateNamed(knowledge.Rule{Phrase: exact}); got != exact {
		t.Errorf("a rule of exactly %d characters is named %q", gateName, got)
	}

	over := gateNamed(knowledge.Rule{Phrase: strings.Repeat("a", gateName+1)})
	if !strings.HasSuffix(over, "…") {
		t.Errorf("a rule one character too long is named %q, with nothing saying it was cut", over)
	}

	if len([]rune(over)) > gateName+1 {
		t.Errorf("the name is %d characters, want no more than %d and the mark", len([]rune(over)), gateName)
	}

	// The first line, because a rule with its reasoning under it is one
	// rule and the name is the sentence at the top of it.
	if got := gateNamed(knowledge.Rule{
		Phrase: "no UPDATE or DELETE in ledger\n\nbecause the ledger is the account",
	}); got != "no UPDATE or DELETE in ledger" {
		t.Errorf("a rule with its reasoning under it is named %q", got)
	}
}

// TestWhatTheRecordSaysAboutAGateThatRan.
//
// The rule's name beside its sentence, because the sentence is what a model
// reads and the name is what a later reader follows: the name survives
// somebody rewording the rule and the gate's own does not. A gate the flow
// declares has no rule behind it and must carry none, or the name points at
// a rule nobody wrote.
func TestWhatTheRecordSaysAboutAGateThatRan(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "KNOW-GATE-2", "touch the ledger", "quick")
	if err != nil {
		t.Fatal(err)
	}

	fact := stopping("No UPDATE or DELETE in ledger.", "exit 3")
	fact.Scope = knowledge.Scope{Kind: knowledge.Repo, Repo: r.Path}

	if _, err := knowledge.NewStore(s.Root()).Save(fact); err != nil {
		t.Fatalf("save the fact: %v", err)
	}

	// The name it was given when it was written down, which is what the
	// record has to point at: it survives somebody rewording the sentence.
	filed, err := knowledge.NewStore(s.Root()).LoadRepo(r.Path)
	if err != nil || len(filed) != 1 {
		t.Fatalf("read the fact back: %d rules, %v", len(filed), err)
	}

	saved := filed[0].ID

	// A gate the flow declares, beside the one the rule brought, and a
	// phase number that is not the first.
	const phaseNumber = 3

	own := flow.Phase{Name: "implement", Gates: []flow.Gate{{Name: "build", Command: "true"}}}

	_, err = runGates(context.Background(), s, tk, own, phaseNumber, t.TempDir(), engine.Result{})
	if err != nil {
		t.Fatalf("runGates: %v", err)
	}

	byGate := map[string]record.Event{}

	for _, e := range mustEvents(t, s, tk) {
		if e.Kind == record.GatePassed || e.Kind == record.GateFailed {
			byGate[e.Data["gate"]] = e
		}
	}

	if len(byGate) != 2 {
		t.Fatalf("%d gates were written down, want the flow's own and the rule's", len(byGate))
	}

	declared, there := byGate["build"]
	if !there {
		t.Fatalf("the gate the flow declares is not in the record: %v", byGate)
	}

	if got, said := declared.Data["rule"]; said {
		t.Errorf("a gate the flow declares says it came from the rule %q", got)
	}

	if declared.Data["n"] != strconv.Itoa(phaseNumber) {
		t.Errorf("the gate says it ran in phase %q, want %d", declared.Data["n"], phaseNumber)
	}

	var fromARule record.Event

	for name, e := range byGate {
		if name != "build" {
			fromARule = e
		}
	}

	if fromARule.Data["rule"] != saved {
		t.Errorf("the gate a rule brought says it came from %q, want %q", fromARule.Data["rule"], saved)
	}

	if fromARule.Data["n"] != strconv.Itoa(phaseNumber) {
		t.Errorf("it says it ran in phase %q, want %d", fromARule.Data["n"], phaseNumber)
	}
}

// TestAGateThatPrintedTooMuchSaysHowMuchThereWas.
//
// A gate's output is whatever the command wrote, and a test suite can print
// megabytes. What is kept is cut, and the count of what there was is the
// only thing telling a reader the tail they are looking at is a tail — a
// gate nothing was cut from must not carry a number they would compare
// against the text and find agrees.
func TestAGateThatPrintedTooMuchSaysHowMuchThereWas(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "KNOW-GATE-3", "print a great deal", "quick")
	if err != nil {
		t.Fatal(err)
	}

	loud := flow.Phase{Name: "implement", Gates: []flow.Gate{
		{Name: "quiet", Command: "echo ok"},
		// One byte over what is kept, written without holding it in a
		// shell variable.
		{Name: "loud", Command: "head -c " + strconv.Itoa(maxOutput+1) + " /dev/zero | tr '\\0' 'a'"},
	}}

	_, err = runGates(context.Background(), s, tk, loud, 1, t.TempDir(), engine.Result{})
	if err != nil {
		t.Fatalf("runGates: %v", err)
	}

	byGate := map[string]record.Event{}

	for _, e := range mustEvents(t, s, tk) {
		if e.Kind == record.GatePassed || e.Kind == record.GateFailed {
			byGate[e.Data["gate"]] = e
		}
	}

	if got, there := byGate["quiet"].Data["bytes"]; there {
		t.Errorf("a gate nothing was cut from says %q bytes were", got)
	}

	if byGate["loud"].Data["bytes"] != strconv.Itoa(maxOutput+1) {
		t.Errorf("a gate one byte over says %q bytes", byGate["loud"].Data["bytes"])
	}

	if kept := len(byGate["loud"].Text); kept > maxOutput+64 {
		t.Errorf("what was kept is %d bytes, which is not a cut at %d", kept, maxOutput)
	}
}
