package ui

// Which cell of a strip a click landed on, swept across every column.
//
// A tab strip and a row of pills are both the same shape: bands laid side by
// side, each claiming a stretch of the row. What a reader notices when the
// arithmetic is one out is that a click on the edge of a tab opens the one
// beside it — so these walk every column of the row rather than the three
// somebody picked.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/settings"
)

// TestEveryColumnOfTheTabStripBelongsToOneTabAtMost.
//
// Two tabs that both claim a column is a click whose answer depends on the
// order they happen to be walked in, and a gap between them is a column that
// looks like a tab and does nothing.
func TestEveryColumnOfTheTabStripBelongsToOneTabAtMost(t *testing.T) {
	m := openOn(t, "ACME-1")

	tabs := m.placeTabs()
	if len(tabs) == 0 {
		t.Fatal("the strip has no tabs to click on")
	}

	for x := -2; x < m.frame.Body.W+4; x++ {
		got := m.hitTabs(x)

		claimed := 0

		for _, one := range tabs {
			if x >= one.x && x < one.x+one.w {
				claimed++
			}
		}

		if claimed > 1 {
			t.Errorf("column %d is claimed by %d tabs at once", x, claimed)
		}

		if claimed == 0 && got.Kind != point.None {
			t.Errorf("column %d is in no tab and answered %+v", x, got)
		}

		if claimed == 1 && got.Kind != point.PaneTab {
			t.Errorf("column %d is in a tab and answered %+v", x, got)
		}
	}

	// Every tab is reachable: one drawn where nothing can click it is a tab
	// a reader can see and not open.
	for i, one := range tabs {
		for _, x := range []int{one.x, one.x + one.w - 1} {
			got := m.hitTabs(x)
			if got.Kind != point.PaneTab || got.Pane != int(one.tab) {
				t.Errorf("tab %d at column %d answered %+v", i, x, got)
			}
		}

		// And it does not reach past its own width.
		if before := m.hitTabs(one.x - 1); before.Kind == point.PaneTab && before.Pane == int(one.tab) {
			t.Errorf("tab %d answers the column before it", i)
		}

		if after := m.hitTabs(one.x + one.w); after.Kind == point.PaneTab && after.Pane == int(one.tab) {
			t.Errorf("tab %d answers the column after it", i)
		}
	}
}

// TestATabStripLeavesNoGapBetweenItsTabs, which is the other half: the gaps
// are the separator the strip is drawn with, and a tab that started one
// column late would put a dead cell on the edge of every one of them.
func TestATabStripLeavesNoGapBetweenItsTabs(t *testing.T) {
	m := openOn(t, "ACME-1")

	tabs := m.placeTabs()
	if len(tabs) < 2 {
		t.Skip("one tab has no gap to leave")
	}

	for i := 1; i < len(tabs); i++ {
		gap := tabs[i].x - (tabs[i-1].x + tabs[i-1].w)
		if gap < 0 {
			t.Errorf("tab %d starts %d columns inside the one before it", i, -gap)
		}

		if gap > len(tabGap) {
			t.Errorf("tab %d leaves %d columns of nothing before it, and the separator is %d wide",
				i, gap, len(tabGap))
		}
	}

	// The strip starts one column in, where the border is drawn.
	if tabs[0].x != 1 {
		t.Errorf("the strip opens at column %d", tabs[0].x)
	}

	// And every tab has a width somebody can hit.
	for i, one := range tabs {
		if one.w < 1 {
			t.Errorf("tab %d is %d columns wide", i, one.w)
		}
	}
}

// TestEveryColumnOfASettingsRowBelongsToOnePillAtMost.
//
// The pills are a row of bands like the tabs, and they are wider than they
// look: a pill carries a mark when it is the chosen one, so choosing moves
// every pill to its right. A click on the edge of one that answered its
// neighbour would change a setting the reader did not point at.
func TestEveryColumnOfASettingsRowBelongsToOnePillAtMost(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenSettings
	m = unfolded(m)

	rows := m.settingRowsList()
	if len(rows) == 0 {
		t.Fatal("the settings screen has no rows to click on")
	}

	// The first row's own line, which is where its pills are drawn.
	y := m.frame.Body.Y + settingsLine(t, m, rows[0].Key)

	for x := 0; x < m.frame.Body.W+4; x++ {
		got := m.hitSettings(x, y)

		if got.Kind == point.None {
			continue
		}

		if got.Kind != point.SettingsRow {
			t.Errorf("column %d of a settings row answered %+v", x, got)
			continue
		}

		if got.Pane < 0 || got.Pane >= len(rows) {
			t.Errorf("column %d answered row %d of %d", x, got.Pane, len(rows))
		}

		// A column left of where the pills start is the row and never one
		// of its options: the name of a setting is not a way to change it.
		if x < settings.PillsAt && got.Field != "" {
			t.Errorf("column %d is left of the pills and answered the option %q", x, got.Field)
		}

		// And an option it answers is one this row actually offers.
		if got.Field != "" {
			offered := false

			for _, opt := range rows[got.Pane].Options {
				if opt == got.Field {
					offered = true
				}
			}

			if !offered {
				t.Errorf("column %d answered the option %q, which row %d does not offer",
					x, got.Field, got.Pane)
			}
		}
	}

	// Every option of the first row is reachable, and no two share a
	// column: a pill drawn where nothing can click it is a setting a reader
	// can see and not choose.
	at := map[string]int{}

	for x := 0; x < m.frame.Body.W+4; x++ {
		if got := m.hitSettings(x, y); got.Kind == point.SettingsRow && got.Field != "" {
			at[got.Field]++
		}
	}

	for _, opt := range rows[0].Options {
		if at[opt] == 0 {
			t.Errorf("the option %q is drawn where nothing can click it", opt)
		}
	}
}
