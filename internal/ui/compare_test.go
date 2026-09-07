package ui

// Running the flow's own checks on both sides of the change, and saying
// which of them answered differently.

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// TestTheChecksAreTheFlowsOwnAndNothingInvented. A command guessed from the
// shape of the repository — a go.mod here, a package.json there — is Orbit
// deciding what this project's tests are, and being wrong about it costs ten
// minutes and prints something nobody asked for.
func TestTheChecksAreTheFlowsOwnAndNothingInvented(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// A task with no flow, and a window with nowhere to read flows from,
	// both answer with nothing rather than a guess.
	if got := m.checksOf(view.Task{ID: "ACME-1"}); len(got) != 0 {
		t.Errorf("a task with no flow offered %+v", got)
	}

	m.opts.Flows = nil
	if got := m.checksOf(view.Task{ID: "ACME-1", Flow: "careful"}); len(got) != 0 {
		t.Errorf("a window with no flows offered %+v", got)
	}
}

// TestALoopsChecksAreChecksToo, and so are the ones inside the phases it
// repeats: what the loop stops on is the flow's own statement of done.
func TestALoopsChecksAreChecksToo(t *testing.T) {
	got := checksIn(flow.Phase{
		Name:  "1-fix",
		Gates: []flow.Gate{{Name: "vet", Command: "go vet ./..."}},
		Loop: &flow.Loop{
			Until:  []flow.Gate{{Name: "tests", Command: "go test ./..."}},
			Phases: []flow.Phase{{Gates: []flow.Gate{{Name: "lint", Command: "golangci-lint run"}}}},
		},
	})

	var named []string
	for _, c := range got {
		named = append(named, c.Name)
	}

	for _, want := range []string{"vet", "tests", "lint"} {
		if !strings.Contains(strings.Join(named, " "), want) {
			t.Errorf("the checks are %v, want %q among them", named, want)
		}
	}
}

// TestOnlyTheChecksThatAnsweredDifferentlyAreTheReading. A check that fails
// on both sides was already failing, and one that passes on both is not news.
func TestOnlyTheChecksThatAnsweredDifferentlyAreTheReading(t *testing.T) {
	got := diverged([]repo.Divergence{
		{Check: repo.Check{Name: "same"}, Base: repo.Ran{Exit: 0}, Now: repo.Ran{Exit: 0}},
		{Check: repo.Check{Name: "broke"}, Base: repo.Ran{Exit: 0}, Now: repo.Ran{Exit: 1}},
		{Check: repo.Check{Name: "fixed"}, Base: repo.Ran{Exit: 1}, Now: repo.Ran{Exit: 0}},
	})

	if len(got) != 2 {
		t.Fatalf("%d of three checks answered differently: %+v", len(got), got)
	}

	for _, d := range got {
		if d.Name == "same" {
			t.Error("a check that answered the same is in the reading")
		}
	}
}

// TestTheAnswerLandsOnTheTaskItWasAskedAbout. The reader may have moved on
// by the time the checks finish, and writing it into whatever is on screen
// would put one task's checks under another's name.
func TestTheAnswerLandsOnTheTaskItWasAskedAbout(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", fixtureEntries())
	m.weigh.running = true

	elsewhere := m.tookComparison(comparedMsg{id: "ACME-9999", got: []repo.Divergence{
		{Check: repo.Check{Name: "broke"}, Base: repo.Ran{Exit: 0}, Now: repo.Ran{Exit: 1}},
	}})
	if elsewhere.weigh.checksKnown {
		t.Error("an answer about another task was written into this one")
	}

	if elsewhere.weigh.running {
		t.Error("the answer landing did not stop the wait")
	}

	// The task on screen is written in, and said out loud.
	landed := m.tookComparison(comparedMsg{id: "ACME-2662", got: []repo.Divergence{
		{Check: repo.Check{Name: "broke"}, Base: repo.Ran{Exit: 0}, Now: repo.Ran{Exit: 1}},
		{Check: repo.Check{Name: "same"}, Base: repo.Ran{Exit: 0}, Now: repo.Ran{Exit: 0}},
	}})
	if !landed.weigh.checksKnown {
		t.Fatal("the answer about the task on screen was dropped")
	}

	if !strings.Contains(landed.message, "1 of 2") {
		t.Errorf("the band says %q, want it to count what diverged", landed.message)
	}

	// And a run that could not happen says why rather than showing nothing.
	failed := m.tookComparison(comparedMsg{id: "ACME-2662", err: errNoSupervisor})
	if !strings.Contains(failed.message, "could not be run") {
		t.Errorf("a failed comparison says %q", failed.message)
	}
}

// TestMovingToAnotherTaskForgetsTheComparison, because it was about the one
// that was on screen.
func TestMovingToAnotherTaskForgetsTheComparison(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", fixtureEntries())
	m = m.tookComparison(comparedMsg{id: "ACME-2662", got: []repo.Divergence{
		{Check: repo.Check{Name: "broke"}, Base: repo.Ran{Exit: 0}, Now: repo.Ran{Exit: 1}},
	}})

	gone := m.forgetComparison()
	if gone.weigh.checksKnown || len(gone.weigh.checks) != 0 {
		t.Errorf("moving away kept %+v", gone.weigh)
	}
}

// TestTheImpactPaneOffersToRunTheChecksAndSaysWhileItIs.
func TestTheImpactPaneOffersToRunTheChecksAndSaysWhileItIs(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", fixtureEntries())

	offered := ansi.Strip(strings.Join(m.compareOffer(), "\n"))
	if strings.TrimSpace(offered) == "" {
		t.Error("the pane offers nothing at all")
	}

	m.weigh.running = true
	m.weigh.since = m.now.Add(-42 * time.Second)

	if got := m.comparedFor(); got != 42*time.Second {
		t.Errorf("the run has been out %v, want 42s", got)
	}

	// With nothing out, there is no count to make.
	idle, _ := openWith(t, "ACME-2662", fixtureEntries())
	if got := idle.comparedFor(); got != 0 {
		t.Errorf("a comparison nobody started has been out %v", got)
	}
}
