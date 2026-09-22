package ui

// A file on the map, pointed at: its diff.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/panes"
	"github.com/e1i0r/orbit/internal/ui/patch"
)

// mapFileAt is which file one row of the map pane is, counting the scroll
// the reader has done.
func (m Model) mapFileAt(row int) (string, bool) {
	if m.tab != tabMap || row < 0 {
		return "", false
	}

	file, on := panes.MapFiles(m.paneEnv(tabMap))[row+m.panes[tabMap].YOffset()]

	return file, on
}

// showDiffOf puts the diff on top, scrolled to where one file begins.
//
// The map says where the task went and the diff says what it did there, so
// a file on the map is a way into its diff. A file the diff does not carry
// yet, because it has not been read, is said rather than jumped to.
func (m Model) showDiffOf(file string) Model {
	m = m.showTab(tabDiff)

	for _, f := range patch.Files(strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")) {
		if f.Path == file {
			m.panes[tabDiff].SetYOffset(f.StartLine)
			m.diffFilePicker = false

			return m.syncPanes()
		}
	}

	return m.say(m.opts.Words.T("map.no_diff_for", "{file} is not in the diff yet",
		about("file", file)))
}
