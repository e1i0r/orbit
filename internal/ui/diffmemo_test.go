package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// TestTheDiffIsDrawnAgainWhenWhatItIsDrawnFromChanges. The diff pane is
// kept between draws; what matters is that nothing it is drawn from can
// change without the pane changing with it — a fold, the width, the text,
// the language — and that an unchanged pane is the same drawing.
func TestTheDiffIsDrawnAgainWhenWhatItIsDrawnFromChanges(t *testing.T) {
	m, _ := openIn(t, words.For("en"), "ACME-2698", fixtureEntries(), wideDiff())
	m = showing(t, m, tabDiff)

	was, _ := m.diffRows()
	if again, _ := m.diffRows(); !slices.Equal(was, again) {
		t.Fatal("the same diff was drawn two ways")
	}

	for name, change := range map[string]func(Model) Model{
		"a fold":    func(m Model) Model { return m.toggleCollapseAll() },
		"the width": func(m Model) Model { return m.resize(m.width-20, m.height) },
		"the text": func(m Model) Model {
			m.diff = strings.Replace(m.diff, "+", "+changed ", 1)
			return m
		},
		"the language": func(m Model) Model { return m.language("es") },
	} {
		got, _ := change(m).diffRows()
		if slices.Equal(was, got) {
			t.Errorf("after %s the diff was the drawing from before", name)
		}

		// And back to the original, which is drawn as it was.
		if back, _ := m.diffRows(); !slices.Equal(was, back) {
			t.Errorf("after %s the original diff was not drawn as it was", name)
		}
	}
}
