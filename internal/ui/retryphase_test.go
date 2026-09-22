package ui

// Which phase the retry offers, and when it offers nothing.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestTheRetryOffersThePhaseThatWentWrong.
func TestTheRetryOffersThePhaseThatWentWrong(t *testing.T) {
	for _, c := range []struct {
		why     string
		record  []view.Entry
		want    string
		offered bool
	}{
		{
			why: "a flow that ran clean has nothing to run again",
			record: []view.Entry{
				{Kind: "phase.started", Phase: "implement"},
				{Kind: "phase.finished", Phase: "implement"},
				{Kind: "phase.started", Phase: "review"},
				{Kind: "phase.finished", Phase: "review"},
			},
		},
		{
			why: "the phase that broke",
			record: []view.Entry{
				{Kind: "phase.started", Phase: "implement"},
				{Kind: "phase.finished", Phase: "implement"},
				{Kind: "phase.started", Phase: "review"},
				{Kind: "phase.failed", Phase: "review"},
			},
			want: "review", offered: true,
		},
		{
			why: "a phase somebody stopped is one to run again too",
			record: []view.Entry{
				{Kind: "phase.started", Phase: "fix"},
				{Kind: "phase.cancelled", Phase: "fix"},
			},
			want: "fix", offered: true,
		},
		{
			// The whole point of the rule: what passed is not re-run, so
			// the phase offered is the one that failed and never an
			// earlier one.
			why: "the later failure, not the earlier one",
			record: []view.Entry{
				{Kind: "phase.started", Phase: "implement"},
				{Kind: "phase.failed", Phase: "implement"},
				{Kind: "phase.started", Phase: "implement"},
				{Kind: "phase.finished", Phase: "implement"},
				{Kind: "phase.started", Phase: "fix"},
				{Kind: "phase.failed", Phase: "fix"},
			},
			want: "fix", offered: true,
		},
		{
			why: "a phase that failed and then ran through is not offered",
			record: []view.Entry{
				{Kind: "phase.started", Phase: "review"},
				{Kind: "phase.failed", Phase: "review"},
				{Kind: "phase.started", Phase: "review"},
				{Kind: "phase.finished", Phase: "review"},
			},
		},
	} {
		got, ok := troubled(c.record)
		if ok != c.offered || got != c.want {
			t.Errorf("%s: troubled = (%q, %v), want (%q, %v)", c.why, got, ok, c.want, c.offered)
		}
	}
}

// TestTheBarOffersTheRetryOnlyWhenThereIsOne.
func TestTheBarOffersTheRetryOnlyWhenThereIsOne(t *testing.T) {
	broke := []view.Entry{
		{Kind: "phase.started", Phase: "review"},
		{Kind: "phase.failed", Phase: "review"},
	}

	m := openOn(t, "PAY-1")
	m.opts.Retry = func(view.Task, string, int) (int, error) { return 1, nil }

	// Nothing wrong: no hint, and pressing it says so rather than running
	// anything.
	if _, can := m.canRetryPhase(); can {
		t.Error("a task with a clean record offers a retry")
	}

	if bar := hintKeys(m.detailHints()); strings.Contains(bar, "ctrl+r") {
		t.Errorf("the bar offers the retry key on a clean task: %s", bar)
	}

	// A phase that broke: offered, and it names that phase.
	m.entries = broke

	phase, can := m.canRetryPhase()
	if !can || phase != "review" {
		t.Fatalf("canRetryPhase on a broken phase = (%q, %v), want review", phase, can)
	}

	if bar := hintKeys(m.detailHints()); !strings.Contains(bar, "ctrl+r") {
		t.Errorf("the bar does not offer the retry key on a task whose phase broke: %s", bar)
	}

	// A window with no port cannot offer it whatever the record says.
	m.opts.Retry = nil
	if _, can := m.canRetryPhase(); can {
		t.Error("a window with no retry port offers a retry")
	}
}

// hintKeys is the keystrokes the bar is carrying, which is the field a
// keypress is matched against. The label beside it is what the reader
// sees, and asserting on that would be a test of the theme.
func hintKeys(hints []barHint) string {
	var keys []string
	for _, h := range hints {
		keys = append(keys, h.key)
	}

	return strings.Join(keys, " ")
}
