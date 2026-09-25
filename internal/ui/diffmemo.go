package ui

// The diff pane, remembered between draws.
//
// Every pane is drawn again on every tick and every key, and the diff is the
// one that costs: a three-hundred-file change took sixty milliseconds a
// draw, most of it laying the cards out, twice a second while it sat on
// screen doing nothing. What it is drawn from changes far less often than
// that — a new read of the worktree, a fold, a resize — so the drawing is
// kept with what it was drawn from, and drawn again when that differs.

import (
	"maps"
	"slices"
	"strings"

	"github.com/e1i0r/orbit/internal/ui/panes"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// diffDrawn is the last drawing of the diff pane and what it was drawn from.
type diffDrawn struct {
	key   diffKey
	lines []string
	heads map[int]int
}

// diffKey is everything panes.Diff reads.
//
// The record's entries are counted rather than compared: they are the
// reasons drawn beside the lines, and the record only ever grows, so a new
// reason is a longer record. Colours are the theme's, which can change
// while the window is open.
type diffKey struct {
	task, lang, theme, failed, diff string
	band                            view.Band
	known, missing, gone, rationale bool
	expanded                        bool
	width, entries                  int
	collapsed                       string
}

// diffKeyOf reads the key off the environment the pane is drawn with.
func diffKeyOf(e panes.Env) diffKey {
	return diffKey{
		task: e.Task.ID, band: view.BandOf(e.Task), lang: e.Words.T("header.lang_badge", "EN"),
		theme: theme.CurrentTheme(), failed: e.DiffFailed, diff: e.Diff,
		known: e.DiffKnown, missing: e.DiffMissing, gone: e.Gone, rationale: e.Rationale,
		expanded: e.Expanded, width: e.Width, entries: len(e.Entries),
		collapsed: strings.Join(shut(e.Collapsed), "\x00"),
	}
}

// shut is the files folded away, in an order two maps with the same files
// in them agree on.
func shut(c map[string]bool) []string {
	var out []string

	for _, f := range slices.Sorted(maps.Keys(c)) {
		if c[f] {
			out = append(out, f)
		}
	}

	return out
}

// same is whether two keys would draw the same pane.
func (k diffKey) same(o diffKey) bool { return k == o }
