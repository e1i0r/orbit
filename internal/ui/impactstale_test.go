package ui

// The one signal that says the impact reading was taken too early, and the
// guard that keeps a second reading from becoming one per poll.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
)

// TestTheReadingIsTakenAgainWhenItDisagreesWithTheDiff. It is taken once
// when the view opens, and a task that is still running had written nothing
// then: a pane saying "this task changed no files" beside a pane showing
// three of them is the window contradicting itself.
func TestTheReadingIsTakenAgainWhenItDisagreesWithTheDiff(t *testing.T) {
	m := reading(t, repo.Impact{})
	m.detail = "ACME-1"

	if m.staleImpact() {
		t.Error("a reading of nothing beside a diff of nothing is stale")
	}

	m.diff = "diff --git a/pricing.py b/pricing.py\n+ changed\n"
	if !m.staleImpact() {
		t.Error("a reading of nothing beside a diff of something is not stale")
	}

	// And the other way round: files read, diff gone.
	m = reading(t, repo.Impact{Changed: []string{"pricing.py"}})
	m.diff = ""

	if !m.staleImpact() {
		t.Error("a reading of files beside an empty diff is not stale")
	}
}

// TestTheReadingIsTakenAgainOnlyOnce. The diff is polled every couple of
// seconds, so a disagreement neither side can settle is not a second reading
// — it is one after every poll, five hundred commits of git log each, for as
// long as the pane is open. It was: the pane spent its life loading, and the
// window went slow with it.
func TestTheReadingIsTakenAgainOnlyOnce(t *testing.T) {
	m := reading(t, repo.Impact{})
	m.detail = "ACME-1"
	m.diff = "diff --git a/pricing.py b/pricing.py\n+ changed\n"

	if !m.staleImpact() {
		t.Fatal("a reading of nothing beside a diff of something is not stale")
	}

	m = m.forgetImpact()
	m.weigh.reread = true
	m = m.tookImpact(impactMsg{id: "ACME-1"})

	if m.staleImpact() {
		t.Error("the same disagreement asked for a third reading")
	}
}
