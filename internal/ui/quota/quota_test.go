package quota

// The quota screen: what it says about each engine, and the one key that
// closes it.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/roster"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// reading is a fixture per engine covering the three answers there are:
// windows, per token, and nowhere to look.
func reading(engine string) roster.Reading {
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

// world is the Env: three engines, one of each shape.
func world(t *testing.T) Env {
	t.Helper()

	return Env{
		Words: words.For("en"),
		Keys:  keymap.New(words.For("en")),
		Engines: func() []roster.Engine {
			return []roster.Engine{{Name: "claude"}, {Name: "codex"}, {Name: "bare"}}
		},
		Read: reading,
	}
}

// TestTheScreenNamesEveryEngineAndWhatItHasLeft. Every engine the picker
// offers is a row here, including the ones with no percentage: an engine
// missing from this list would be read as an engine with nothing to report,
// and those are different sentences.
func TestTheScreenNamesEveryEngineAndWhatItHasLeft(t *testing.T) {
	e := world(t)

	joined := ansi.Strip(strings.Join(View(30, 100, e), "\n"))

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

// TestTheScreenSaysHowToLeave, and leaves only on those keys. Every other
// key does nothing rather than reaching the board underneath it.
func TestTheScreenSaysHowToLeave(t *testing.T) {
	e := world(t)

	joined := ansi.Strip(strings.Join(View(30, 100, e), "\n"))
	if !strings.Contains(joined, e.Keys.Back.Help().Key) {
		t.Errorf("the quota screen does not say which key leaves it:\n%s", joined)
	}

	if out := Key(tea.KeyPressMsg{Code: 'j', Text: "j"}, e); out.Leave {
		t.Error("a key the screen has no use for closed it")
	}

	if out := Key(tea.KeyPressMsg{Code: tea.KeyEscape}, e); !out.Leave {
		t.Error("escape did not close the screen")
	}
}

// TestTheBarIsTheShareThatIsGone. The bar and the sentence beside it are one
// fact, so a window with something spent in it has to have a mark: a bar
// that rounds down shows nothing happening until the sixth percent, on a
// screen whose whole subject is what has been used.
func TestTheBarIsTheShareThatIsGone(t *testing.T) {
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
		if got := strings.Count(ansi.Strip(bar(c.pct, quotaBarCells)), quotaSpent); got != c.spent {
			t.Errorf("a window %.0f%% used drew %d spent cells of %d, want %d",
				c.pct, got, quotaBarCells, c.spent)
		}
	}

	if got := bar(50, 0); got != "" {
		t.Errorf("a bar of no cells = %q, want nothing drawn", got)
	}

	// Spent and left are one bar of a fixed width: a bar that grows with
	// what has been used pushes the sentence beside it off the line, and
	// the line under it reads as a different engine's.
	for _, pct := range []float64{0, 1, 33.3, 50, 74.9, 75, 99.9, 100, 140} {
		if got := len([]rune(ansi.Strip(bar(pct, quotaBarCells)))); got != quotaBarCells {
			t.Errorf("a window %.1f%% used drew a bar of %d cells, want %d",
				pct, got, quotaBarCells)
		}
	}
}

// TestTheBarTurnsAtTheShareWorthNoticing. The colour is the glance and the
// number is the reading, so the two have to change at the same place: a bar
// that stays calm through three quarters of a window and then reports an
// overage is a bar nobody looked at in time.
func TestTheBarTurnsAtTheShareWorthNoticing(t *testing.T) {
	warned := func(pct float64) bool {
		drawn := bar(pct, quotaBarCells)
		spent := strings.Count(ansi.Strip(drawn), quotaSpent)

		return strings.HasPrefix(drawn,
			theme.Paint(theme.Warn).Render(strings.Repeat(quotaSpent, spent)))
	}

	for _, c := range []struct {
		pct  float64
		want bool
	}{
		{1, false},
		{50, false},
		{quotaFull - 0.1, false},
		{quotaFull, true},
		{90, true},
		{140, true},
	} {
		if got := warned(c.pct); got != c.want {
			t.Errorf("a window %.1f%% used drew a warned bar = %v, want %v", c.pct, got, c.want)
		}
	}
}

// TestAScreenWithNoRowsToDrawDrawsNothing. Every window passes through no
// height at all while somebody drags its corner, and a screen that draws its
// title into a body with no room for it is a title the terminal wraps.
func TestAScreenWithNoRowsToDrawDrawsNothing(t *testing.T) {
	e := world(t)

	for _, h := range []int{-1, 0} {
		if got := View(h, 100, e); got != nil {
			t.Errorf("a screen %d rows tall drew %d lines, want none", h, len(got))
		}
	}

	if got := View(1, 100, e); len(got) == 0 {
		t.Error("a screen with a row in it drew nothing")
	}
}

// TestANarrowScreenKeepsTheSentence. The bar is the number at a glance and
// the sentence is the number; where only one fits, the one that survives is
// the one that can be read.
func TestANarrowScreenKeepsTheSentence(t *testing.T) {
	e := world(t)
	claude := reading("claude")

	wide := engineLines(claude, quotaBarFloor, e)
	if !strings.Contains(ansi.Strip(wide[0]), quotaSpent) {
		t.Errorf("a body wide enough for the bar drew none: %q", wide[0])
	}

	narrow := engineLines(claude, quotaBarFloor-1, e)
	if strings.Contains(ansi.Strip(narrow[0]), quotaSpent) {
		t.Errorf("a body too narrow for the bar drew one anyway: %q", narrow[0])
	}

	if !strings.Contains(ansi.Strip(narrow[0]), "12% used in 5h") {
		t.Errorf("the narrow line %q dropped the sentence instead of the bar", narrow[0])
	}
}

// TestAWindowWithNoDoorsReadsNothingRatherThanPanicking, which is the shape
// a window built without a quota port is in.
func TestAWindowWithNoDoorsReadsNothingRatherThanPanicking(t *testing.T) {
	if got := Readings(Env{Words: words.For("en")}); got != nil {
		t.Errorf("a screen with no doors read %+v", got)
	}
}
