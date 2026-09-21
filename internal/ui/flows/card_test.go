package flows

// The card a phase is read as, on the screen that shows what a flow does.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/flow"
)

// loaded is a phase carrying every badge there is.
func loaded() flow.Phase {
	return flow.Phase{
		Name: "reconcile-the-settlements", Engine: "opencode", Model: "claude-opus-5",
		Effort: "high", Thinking: "on", FeedOutput: true, Wait: true,
		Prompt: "go through every unmatched settlement and say why it did not match",
	}
}

// TestACardSaysEverythingThatIsTrueOfItsPhase. Each badge is a fact a
// reader decides on — what runs it, how hard it thinks, whether it is fed
// the last output, whether it stops for a human — and a card that drew them
// only sometimes would be read as a phase that does not do them.
func TestACardSaysEverythingThatIsTrueOfItsPhase(t *testing.T) {
	e := world(t)

	var s State

	said := ansi.Strip(strings.Join(s.phaseCard(2, loaded(), 140, e), "\n"))

	for _, want := range []string{
		"Phase 3",                   // numbered from where it sits
		"reconcile-the-settlements", //
		"opencode/claude-opus-5",    // what runs it
		"effort: high",              //
		"thinking: on",              //
		"feeds output",              //
		"⏸ human gate",              //
		"why it did not match",      // and what it was told
	} {
		if !strings.Contains(said, want) {
			t.Errorf("the card does not say %q:\n%s", want, said)
		}
	}
}

// TestACardSaysNothingThatIsNotTrue. The two dials that have a default say
// nothing while they are on it: a badge for every phase is a badge nobody
// reads.
func TestACardSaysNothingThatIsNotTrue(t *testing.T) {
	e := world(t)

	var s State

	plain := flow.Phase{Name: "implement", Engine: "zeta", Thinking: "adaptive", Effort: "default"}

	said := ansi.Strip(strings.Join(s.phaseCard(0, plain, 140, e), "\n"))

	for _, no := range []string{"effort:", "thinking:", "feeds output", "human gate"} {
		if strings.Contains(said, no) {
			t.Errorf("a phase on its defaults says %q:\n%s", no, said)
		}
	}

	// A model nobody set says so in a word rather than as an empty space.
	if !strings.Contains(said, "zeta/default") {
		t.Errorf("a phase with no model of its own says %q", said)
	}

	// And with nothing to say, it is one line.
	if len(s.phaseCard(0, plain, 140, e)) != 1 {
		t.Errorf("a phase with no prompt and no badges is %d rows", len(s.phaseCard(0, plain, 140, e)))
	}
}

// TestNoRowOfACardRunsPastTheWindow. Five badges is a hundred and twenty-
// five cells, and a card that wraps moves every card under it down — on a
// screen that is read by counting phases.
func TestNoRowOfACardRunsPastTheWindow(t *testing.T) {
	e := world(t)

	var s State

	for _, ph := range []flow.Phase{
		loaded(),
		{Name: "implement", Engine: "zeta"},
		{Name: strings.Repeat("x", 90), Engine: "zeta", Model: "one"},
		{Name: "tests", Engine: "zeta", Loop: &flow.Loop{
			Max: 4, Until: []flow.Gate{{Name: "go test ./..."}},
			Phases: []flow.Phase{{Name: "inner", Engine: "zeta", FeedOutput: true}},
		}},
	} {
		for w := 40; w <= 160; w++ {
			for i, r := range s.phaseCard(0, ph, w, e) {
				if got := lipgloss.Width(r); got > w {
					t.Fatalf("at %d columns, row %d of %q's card is %d cells: %q",
						w, i, ph.Name, got, ansi.Strip(r))
				}
			}
		}
	}
}

// TestAWrappedCardKeepsEveryBadge. Cut, the card loses the badges at the
// end — and the last of them is the one that says the phase stops for a
// human, which is the one a reader is looking for.
func TestAWrappedCardKeepsEveryBadge(t *testing.T) {
	e := world(t)

	var s State

	for _, w := range []int{60, 80, 100} {
		said := strings.Join(strings.Fields(
			ansi.Strip(strings.Join(s.phaseCard(0, loaded(), w, e), " "))), " ")

		for _, want := range []string{"effort: high", "thinking: on", "feeds output", "⏸ human gate"} {
			if !strings.Contains(said, want) {
				t.Errorf("at %d columns the card drops %q:\n%s", w, want, said)
			}
		}
	}
}

// TestALoopCardSaysWhatEndsIt. A loop is not a machine for spending a quota
// window on a wall: the thing that ends it is a command's exit code, and
// that is what a reader most needs to see.
func TestALoopCardSaysWhatEndsIt(t *testing.T) {
	e := world(t)

	var s State

	ph := flow.Phase{Name: "tests", Engine: "zeta", Loop: &flow.Loop{
		Max: 4, Until: []flow.Gate{{Name: "go test ./..."}, {Name: "make lint"}},
		Phases: []flow.Phase{{Name: "fix", Engine: "zeta", Model: "one", FeedOutput: true}},
	}}

	said := ansi.Strip(strings.Join(s.loopCard(1, ph, 140, e), "\n"))

	for _, want := range []string{"Phase 2", "tests", "4", "go test ./...", "make lint", "fix", "zeta/one"} {
		if !strings.Contains(said, want) {
			t.Errorf("the loop's card does not say %q:\n%s", want, said)
		}
	}
}
