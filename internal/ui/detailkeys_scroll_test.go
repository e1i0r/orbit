package ui

// detailkeys_more_coverage_test.go covers logOf's own three answers — no
// reader at all, a reader that failed, and one that answered — and scroll's
// follow/release rule on the timeline tab versus every other pane.

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

func TestLogOfEveryAnswer(t *testing.T) {
	tk := view.Task{ID: "ACME-2662", RepoPath: "/r/payments"}

	// 1. No reader at all.
	msg := logOf(nil, tk)()
	lm, ok := msg.(logMsg)
	if !ok || lm.Err == nil {
		t.Fatalf("logOf(nil, ...) = %#v, want a logMsg carrying an error", msg)
	}

	// 2. A reader that fails.
	r := &fakeReader{logErr: errors.New("record damaged")}
	msg = logOf(r, tk)()
	lm, ok = msg.(logMsg)
	if !ok || lm.Err == nil {
		t.Fatalf("logOf with a failing reader = %#v, want the error carried", msg)
	}

	// 3. A reader that answers.
	r = &fakeReader{entries: fixtureEntries()}
	msg = logOf(r, tk)()
	lm, ok = msg.(logMsg)
	if !ok || lm.Err != nil || len(lm.Entries) == 0 {
		t.Fatalf("logOf with a working reader = %#v, want the entries carried with no error", msg)
	}
}

// TestScrollFollowReleaseOnTheTimeline is the one rule scroll owns: the
// timeline tab arms following when a scroll reaches the bottom and releases
// it the moment a scroll moves the view upward — and no other tab is
// affected by either.
func TestScrollFollowReleaseOnTheTimeline(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", longLog())
	m = showing(t, m, tabTimeline)
	m.following = false

	// Scrolling to the very bottom re-arms following.
	m.panes[tabTimeline].GotoBottom()
	got := m.scroll(tea.KeyPressMsg{Code: tea.KeyDown})
	if !got.following {
		t.Error("scrolling to the bottom of the timeline did not re-arm following")
	}

	// Scrolling up releases it.
	got = got.scroll(tea.KeyPressMsg{Code: tea.KeyUp})
	if got.following {
		t.Error("scrolling up on the timeline did not release following")
	}

	// The same scroll up on a different tab never touches following at all.
	m2 := showing(t, m, tabOverview)
	m2.following = true
	got2 := m2.scroll(tea.KeyPressMsg{Code: tea.KeyUp})
	if !got2.following {
		t.Error("scrolling on a pane other than the timeline changed following")
	}

	// PageUp, PageDown, First (g, GotoTop) and Last (end, GotoBottom) all
	// move the pane; an unmatched key leaves it untouched.
	_ = m.scroll(tea.KeyPressMsg{Code: tea.KeyPgUp})
	_ = m.scroll(tea.KeyPressMsg{Code: tea.KeyPgDown})
	_ = m.scroll(tea.KeyPressMsg{Code: 'g', Text: "g"})
	_ = m.scroll(tea.KeyPressMsg{Code: tea.KeyEnd})

	before := m.panes[m.tab].YOffset()
	after := m.scroll(tea.KeyPressMsg{Code: 'z', Text: "z"})
	if after.panes[after.tab].YOffset() != before {
		t.Error("an unmatched key moved the pane")
	}
}
