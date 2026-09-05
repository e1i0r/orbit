package task

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

const planAnswer = `Looked at the retry path and picked an approach.

## Decisions

- id: exponential-backoff
  scope: internal/task/run.go, internal/task/gate.go
  decision: Retry a failed phase with exponential backoff and jitter.
  rejected: A fixed five-second delay, which stampedes when many tasks fail at once.

- scope: internal/store/settings.go
  decision: Keep the cap in the settings file rather than in each flow.

## Next

Write it.
`

// TestTheDecisionsOfAPlanAreReadOutOfItsAnswer. What a plan decided is the
// part of it worth keeping, and a section of prose nobody parses is a
// decision that lives in a chat window somebody closes.
func TestTheDecisionsOfAPlanAreReadOutOfItsAnswer(t *testing.T) {
	got := decisionsIn(planAnswer)
	if len(got) != 2 {
		t.Fatalf("read %d decisions, want 2: %+v", len(got), got)
	}

	if got[0].ID != "exponential-backoff" {
		t.Errorf("the first decision is called %q, want the id it gave itself", got[0].ID)
	}

	if got[0].Scope != "internal/task/run.go,internal/task/gate.go" {
		t.Errorf("the first decision governs %q, want both paths it named", got[0].Scope)
	}

	if !strings.Contains(got[0].Text, "exponential backoff") || !strings.Contains(got[0].Text, "five-second delay") {
		t.Errorf("the first decision does not carry what was chosen and what was not:\n%s", got[0].Text)
	}

	// An id nobody wrote is made from the decision itself, because a
	// decision a later line cannot point at is a decision nothing can
	// supersede.
	if got[1].ID == "" {
		t.Error("the second decision has no id at all")
	}
}

// TestAnAnswerWithNoDecisionsSectionYieldsNone.
func TestAnAnswerWithNoDecisionsSectionYieldsNone(t *testing.T) {
	if got := decisionsIn("I read the code and it looks fine.\n\n## Next\n\nGo."); len(got) != 0 {
		t.Errorf("read %d decisions out of an answer that made none: %+v", len(got), got)
	}
}

// TestABulletWithNoDecisionLineIsNotADecision. Half an entry is worse than
// none: it puts a line in the record that says a decision was made and never
// says what it was.
func TestABulletWithNoDecisionLineIsNotADecision(t *testing.T) {
	got := decisionsIn("## Decisions\n\n- id: nothing\n  scope: a.go\n")
	if len(got) != 0 {
		t.Errorf("read %+v out of an entry that decided nothing", got)
	}
}

// TestAPlanIsAskedForItsDecisions. The ask is fixed and it is only made of a
// plan: a phase that is implementing does not stop to write minutes.
func TestAPlanIsAskedForItsDecisions(t *testing.T) {
	tk := Task{ID: "ACME-1", Text: "make it idempotent"}

	plan := prompt(tk, flow.Phase{Name: "1-plan", Engine: "fake"}, nil, nil, "", nil)
	if !strings.Contains(plan, "## Decisions") {
		t.Errorf("a plan phase is not asked for its decisions:\n%s", plan)
	}

	work := prompt(tk, flow.Phase{Name: "2-implement", Engine: "fake"}, nil, nil, "", nil)
	if strings.Contains(work, "## Decisions") {
		t.Errorf("a phase that is not planning is asked for decisions anyway:\n%s", work)
	}
}

// TestAPlanThatFinishesWritesItsDecisionsDown.
func TestAPlanThatFinishesWritesItsDecisionsDown(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-19", "decide something", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := flow.Flow{Name: "task", Phases: []flow.Phase{{Name: "1-plan", Engine: "fake"}}}
	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": engine.NewFake(planAnswer)}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	var made []record.Event

	for _, e := range events {
		if e.Kind == record.DecisionMade {
			made = append(made, e)
		}
	}

	if len(made) != 2 {
		t.Fatalf("the record holds %d decisions, want the 2 the plan made: %v", len(made), kindsOf(events))
	}

	if made[0].Data["scope"] != "internal/task/run.go,internal/task/gate.go" {
		t.Errorf("decision.made carries scope %q, want the paths the plan named", made[0].Data["scope"])
	}

	if made[0].Phase != "1-plan" {
		t.Errorf("decision.made belongs to phase %q, want the plan that made it", made[0].Phase)
	}
}

// TestOnlyAPlanWritesDecisions. A review phase repeating the plan's own
// section would write every decision down twice.
func TestOnlyAPlanWritesDecisions(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-20", "decide nothing", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := flow.Flow{Name: "task", Phases: []flow.Phase{{Name: "implement", Engine: "fake"}}}
	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": engine.NewFake(planAnswer)}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	for _, e := range events {
		if e.Kind == record.DecisionMade {
			t.Errorf("a phase that was not planning wrote a decision down: %+v", e)
		}
	}
}
