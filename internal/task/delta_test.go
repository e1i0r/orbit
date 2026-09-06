package task

// The delta spec: what the change asks, what it promises, what it assumed,
// and what it decided against.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
)

// TestTheFourFieldsAreReadOutOfTheAnswer, repeats and all: one change can
// ask two things of its callers.
func TestTheFourFieldsAreReadOutOfTheAnswer(t *testing.T) {
	answer := strings.Join([]string{
		"Did the work.",
		"",
		"## Delta",
		"",
		"needs: amount must be a Decimal, floats are refused",
		"needs: currency must be an ISO 4217 code",
		"guarantees: rounds half-even in every currency",
		"assumes: timestamps arrive in UTC",
		"instead: rounding per currency, dropped for the configuration it would need",
	}, "\n")

	got := deltaIn(answer)
	if got == nil {
		t.Fatal("the section was not read")
	}

	if lines := strings.Split(got["needs"], "\n"); len(lines) != 2 || !strings.Contains(lines[1], "ISO 4217") {
		t.Errorf("needs came back as %q", got["needs"])
	}

	if !strings.Contains(got["instead"], "configuration") {
		t.Errorf("instead came back as %q", got["instead"])
	}
}

// TestAFieldWithNothingTrueToSayIsLeftOut, unlike the story's five: a change
// that adds no precondition has nothing to say under needs, and a line
// invented to fill the shape is worse than the gap.
func TestAFieldWithNothingTrueToSayIsLeftOut(t *testing.T) {
	got := deltaIn("## Delta\n\nguarantees: the total is never negative\n")
	if got == nil {
		t.Fatal("a delta with one field was refused")
	}

	if len(got) != 1 || got["guarantees"] == "" {
		t.Errorf("the delta is %+v", got)
	}
}

// TestAnAnswerWithNoDeltaWritesNothing, rather than an event with four empty
// fields that a pane would draw as an empty promise.
func TestAnAnswerWithNoDeltaWritesNothing(t *testing.T) {
	if got := deltaIn("Did the work. No section here.\n\n## Story\n\nentry: the API\n"); got != nil {
		t.Errorf("a delta was read out of an answer that has none: %+v", got)
	}
}

// TestTheLastPhaseIsAskedForIt, where the story is asked: a phase in the
// middle of a flow does not know what the task ended up asking of anybody.
func TestTheLastPhaseIsAskedForIt(t *testing.T) {
	if !strings.Contains(deltaAsk, "instead") || !strings.Contains(deltaAsk, "dies with this run") {
		t.Error("the ask does not say why the discarded alternatives matter")
	}
}

// TestEveryPhaseIsAskedForItsDelta. Asked of the last phase alone, a flow
// that ends at a human gate had nothing to show until somebody resumed it —
// and the phase that did the work had already answered and gone.
func TestEveryPhaseIsAskedForItsDelta(t *testing.T) {
	f := flow.Flow{Name: "three", Phases: []flow.Phase{
		{Name: "1-implement", Engine: "claude"},
		{Name: "2-fix", Engine: "claude"},
		{Name: "3-review", Engine: "claude", Wait: true},
	}}

	tk := Task{ID: "ACME-1", Text: "do the thing"}

	for n := 1; n <= len(f.Phases); n++ {
		asked := promptFor(tk, f, n, nil, nil, nil, "", nil)
		if !strings.Contains(asked, "## Delta") {
			t.Errorf("phase %d is not asked for a delta", n)
		}
	}

	// The story is still the last phase's alone: it is about the task, and a
	// phase in the middle does not know how it ends.
	if strings.Contains(promptFor(tk, f, 1, nil, nil, nil, "", nil), "## Story") {
		t.Error("the first phase is asked for the story")
	}
}
