package ui

// quota_test.go is the quota screen: the chip that opens it, what it says
// about each engine, and the one key that closes it.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// quotaFixture is a reading per engine covering the three answers there are:
// windows, per token, and nowhere to look.
func quotaFixture(engine string) QuotaReading {
	switch engine {
	case "claude":
		return QuotaReading{Engine: engine, Sourced: true, Windows: []QuotaWindow{
			{Label: "5h", Pct: 12, ResetsIn: 75 * time.Minute},
			{Label: "week", Pct: 84, ResetsIn: 3 * time.Hour},
		}}
	case "codex":
		return QuotaReading{Engine: engine, Money: true}
	}

	return QuotaReading{Engine: engine}
}

// quotaModel is the window on the quota screen with that reading in it.
func quotaModel(t *testing.T, w, h int) Model {
	t.Helper()

	m, _ := testModel(t, w, h)
	m.opts.Engines = enginesTestList
	m.opts.Quota = quotaFixture

	return m.openQuota()
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

	got := m.hitHeader(x, m.frame.Header.Y)
	if got.Kind != TargetHeaderField || got.Field != "quota" {
		t.Fatalf("hitHeader on the quota chip = %+v, want the quota field", got)
	}

	after, _ := m.leftClick(got)

	opened := asModel(t, after)
	if opened.screen != screenQuota {
		t.Errorf("screen after clicking the chip = %v, want the quota screen", opened.screen)
	}
}

// TestTheQuotaScreenNamesEveryEngineAndWhatItHasLeft. Every engine the
// picker offers is a row here, including the ones with no percentage: an
// engine missing from this list would be read as an engine with nothing to
// report, and those are different sentences.
func TestTheQuotaScreenNamesEveryEngineAndWhatItHasLeft(t *testing.T) {
	m := quotaModel(t, 100, 30)

	joined := ansi.Strip(strings.Join(m.quotaRows(30, 100), "\n"))

	for _, want := range []string{
		"CLAUDE",
		"12% used in 5h · resets in 1h15m",
		"84% used in week · resets in 3h0m",
		"CODEX",
		"billed per token",
		"BARE",
		"no quota source for bare",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("the quota screen does not say %q:\n%s", want, joined)
		}
	}
}

// TestTheQuotaScreenSaysHowToLeave, and leaves only on that key. Every other
// key does nothing rather than reaching the board underneath it.
func TestTheQuotaScreenSaysHowToLeave(t *testing.T) {
	m := quotaModel(t, 100, 30)

	joined := ansi.Strip(strings.Join(m.quotaRows(30, 100), "\n"))
	if !strings.Contains(joined, m.keys.Back.Help().Key) {
		t.Errorf("the quota screen does not say which key leaves it:\n%s", joined)
	}

	stayed, _ := m.quotaKey(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if got := asModel(t, stayed); got.screen != screenQuota {
		t.Errorf("screen after a key the screen has no use for = %v, want it still up", got.screen)
	}

	left, _ := m.quotaKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := asModel(t, left); got.screen != screenList {
		t.Errorf("screen after escape = %v, want the board", got.screen)
	}
}

// TestTheQuotaBarIsTheShareThatIsGone. The bar and the sentence beside it are
// one fact, so a window with something spent in it has to have a mark: a bar
// that rounds down shows nothing happening until the sixth percent, on a
// screen whose whole subject is what has been used.
func TestTheQuotaBarIsTheShareThatIsGone(t *testing.T) {
	for _, c := range []struct {
		pct   float64
		spent int
	}{
		{0, 0},
		{1, 1},
		{50, 8},
		{100, 16},
		{140, 16},
	} {
		got := strings.Count(ansi.Strip(quotaBar(c.pct, quotaBarCells)), quotaSpent)
		if got != c.spent {
			t.Errorf("a window %.0f%% used drew %d spent cells of %d, want %d",
				c.pct, got, quotaBarCells, c.spent)
		}
	}

	if got := quotaBar(50, 0); got != "" {
		t.Errorf("a bar of no cells = %q, want nothing drawn", got)
	}
}

// TestANarrowQuotaScreenKeepsTheSentence. The bar is the number at a glance
// and the sentence is the number; where only one fits, the one that survives
// is the one that can be read.
func TestANarrowQuotaScreenKeepsTheSentence(t *testing.T) {
	m := quotaModel(t, 100, 30)
	reading := quotaFixture("claude")

	wide := m.quotaEngineLines(reading, quotaBarFloor)
	if !strings.Contains(ansi.Strip(wide[0]), quotaSpent) {
		t.Errorf("a body wide enough for the bar drew none: %q", wide[0])
	}

	narrow := m.quotaEngineLines(reading, quotaBarFloor-1)
	if strings.Contains(ansi.Strip(narrow[0]), quotaSpent) {
		t.Errorf("a body too narrow for the bar drew one anyway: %q", narrow[0])
	}

	if !strings.Contains(ansi.Strip(narrow[0]), "12% used in 5h") {
		t.Errorf("the narrow line %q dropped the sentence instead of the bar", narrow[0])
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
	if got := asModel(t, opened); got.screen != screenQuota {
		t.Errorf("screen after %q = %v, want the quota screen",
			m.keys.Quota.Keys()[0], got.screen)
	}
}
