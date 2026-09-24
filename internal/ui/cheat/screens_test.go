package cheat

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/words"
)

// TestTheSheetNamesEveryScreensKey. The sheet still gave the engine dial
// as M after it moved to E, and left out F, K and ^R: it kept a copy of
// the keys, and a copy stops being true the day one of them moves.
func TestTheSheetNamesEveryScreensKey(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	e := world(t, words.For("en"))
	sheet := ansi.Strip(strings.Join(Open(0).View(200, 120, e), "\n"))

	for _, b := range []key.Binding{
		e.Keys.Autopilot, e.Keys.EngineKnobs, e.Keys.Flows, e.Keys.Knowledge,
		e.Keys.Supervisor, e.Keys.Repos, e.Keys.Quota, e.Keys.Language, e.Keys.RetryPhase,
	} {
		if glyph := "[" + b.Help().Key + "]"; !strings.Contains(sheet, glyph) {
			t.Errorf("the sheet does not name %s, which %s", glyph, b.Help().Desc)
		}
	}

	if strings.Contains(sheet, "[M] / 🧠") {
		t.Error("the sheet still gives the engine dial as M")
	}
}
