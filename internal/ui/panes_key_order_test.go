package ui

// The order the tab strip draws its keys in.
//
// It is the const block's order and nothing else, which is easy to get
// wrong by adding a pane in the wrong place: the map went in before the
// diff and the strip drew 9 l 0 i, so a reader counting along the row hit
// a letter where the last digit should have been.

import (
	"strings"
	"testing"
)

// TestTheTabStripCountsBeforeItSpells reads the strip the reader sees.
//
// It walked paneKey over the tab constants the first time, which is a
// different list: the order the strip draws is tabNames', and the two are
// kept the same by hand. So the constants were put right, the test went
// green, and the strip on screen still read 9 l 0 i.
//
// Reading the drawn row is the only assertion that could have caught that,
// and it is the one a reader could have made by looking.
func TestTheTabStripCountsBeforeItSpells(t *testing.T) {
	m := openOn(t, "PAY-1")

	var keys []string

	for _, tab := range m.tabNames() {
		if k := paneKey(tab.tab); k != "" {
			keys = append(keys, k)
		}
	}

	got := strings.Join(keys, "")
	want := "1234567890liwuy"

	if got != want {
		t.Errorf("the strip draws [%s], want [%s] — the digits in order, 0 last of them,\n"+
			"then the letters. The order is tabNames' in panes.go, not the const block's.", got, want)
	}
}

// TestEveryPaneKeyOpensTheePaneItIsDrawnOn: the strip and the keyboard
// cannot disagree, whatever the order is.
func TestEveryPaneKeyOpensTheePaneItIsDrawnOn(t *testing.T) {
	for i := range int(tabCount) {
		key := paneKey(tab(i))
		if key == "" {
			continue
		}

		back, ok := keyToPane(key)
		if !ok {
			t.Errorf("pane %d is drawn with the key %q and nothing answers it", i, key)
			continue
		}

		if back != tab(i) {
			t.Errorf("pane %d is drawn with %q, which opens pane %d", i, key, back)
		}
	}
}
