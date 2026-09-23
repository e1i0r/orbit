package ui

// The bar on a screen of its own offers the way back and nothing of the
// board's.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/ui/point"
)

// TestAScreenOfItsOwnOffersOnlyTheWayBack. The board's keys were drawn on
// every screen: [x] on settings reset a setting, [m] and [?] did nothing,
// and on the supervisor a click typed its letter into the line. The bar now
// answers one key there, esc, and a click on it goes back.
func TestAScreenOfItsOwnOffersOnlyTheWayBack(t *testing.T) {
	for _, c := range []struct {
		what string
		open func(Model) Model
		want screen
	}{
		{"settings", Model.openSettings, screenSettings},
		{"flows", Model.openFlows, screenFlows},
		{"the quota screen", Model.openQuota, screenQuota},
		{"what Orbit knows", Model.openKnowledge, screenKnowledge},
		{"the engine knobs", Model.openEngines, screenEngines},
		{"the repository list", Model.openRepos, screenRepos},
		{"the cheat sheet", Model.openHelp, screenHelp},
		{"the supervisor", Model.openSupervisor, screenSupervisor},
		{"the form", Model.openCompose, screenCompose},
	} {
		m, _ := testModel(t, 100, 30)
		m.opts.Engines = enginesTestList
		m.opts.Quota = quotaFixture

		m = c.open(m)
		if m.screen != c.want {
			t.Fatalf("%s opened on %v, want %v", c.what, m.screen, c.want)
		}

		y := m.frame.BarLineY()

		var back int

		for x := range m.width {
			hit := m.hit(x, y)
			if hit.Kind != point.BarHint {
				continue
			}

			if hit.Key != "esc" {
				t.Errorf("%s: column %d of the bar sends %q, want only esc", c.what, x, hit.Key)

				continue
			}

			back = x
		}

		if back == 0 {
			t.Errorf("%s: nothing on the bar goes back", c.what)

			continue
		}

		if left := clicked(t, m, m.hit(back, y)); left.screen == c.want {
			t.Errorf("%s: a click on [esc] left the screen where it was", c.what)
		}
	}
}
