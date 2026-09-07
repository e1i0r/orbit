package ui

// The quota screen as the window reaches it: the chip that opens it and the
// key that does. What the screen says is tested in internal/ui/quota.

import (
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/roster"
)

// quotaFixture is a reading per engine covering the three answers there are:
// windows, per token, and nowhere to look.
func quotaFixture(engine string) roster.Reading {
	switch engine {
	case "claude":
		return roster.Reading{Engine: engine, Sourced: true, Windows: []roster.Window{
			{Label: "5h", Pct: 12, ResetsIn: 75 * time.Minute},
			{Label: "week", Pct: 84, ResetsIn: 3 * time.Hour},
		}}
	case "codex":
		return roster.Reading{Engine: engine, Money: true}
	}

	return roster.Reading{Engine: engine}
}

// TestTheQuotaChipOpensTheScreenItIsAShortVersionOf. The chip is one
// engine's share of one window at the width a header has for it; the screen
// is every engine's every window. Unnamed, the chip was drawn and answered
// nothing, and the reader who pointed at the number got no more of it.
func TestTheQuotaChipOpensTheScreenItIsAShortVersionOf(t *testing.T) {
	m, _ := testModel(t, 150, 30)
	m.opts.Engines = enginesTestList
	m.opts.Quota = quotaFixture

	x := headerCell(t, m, "⏳")

	got := m.hitHeader(x, m.frame.HeaderLineY())
	if got.Kind != point.HeaderField || got.Field != "quota" {
		t.Fatalf("hitHeader on the quota chip = %+v, want the quota field", got)
	}

	after, _ := m.leftClick(got)

	opened := asModel(t, after)
	if opened.screen != screenQuota {
		t.Errorf("screen after clicking the chip = %v, want the quota screen", opened.screen)
	}
}

// TestTheQuotaScreenIsReachableWithoutAMouse. The chip is the way it was
// asked for, and a chip is a click; every other screen behind a header field
// also has a key, and a reader working from the keyboard should not have to
// learn that this one is the exception.
func TestTheQuotaScreenIsReachableWithoutAMouse(t *testing.T) {
	m, _ := testModel(t, 150, 30)
	m.opts.Engines = enginesTestList
	m.opts.Quota = quotaFixture

	opened, _ := m.Update(press(m.keys.Quota.Keys()[0]))

	shown := asModel(t, opened)
	if shown.screen != screenQuota {
		t.Fatalf("screen after %q = %v, want the quota screen", m.keys.Quota.Keys()[0], shown.screen)
	}

	// And the key that leaves it goes back to the board rather than out of
	// the window.
	left, _ := shown.quotaKey(press("esc"))
	if got := asModel(t, left); got.screen != screenList {
		t.Errorf("screen after escape = %v, want the board", got.screen)
	}
}
