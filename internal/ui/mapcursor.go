package ui

// The map's cursor: the arrows walk its files and ↵ opens the diff of the
// one they stand on. A file opened its diff by a click and by nothing else.

import (
	"sort"

	"github.com/e1i0r/orbit/internal/ui/panes"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// mapStop is one file of the map and the row it is drawn on.
type mapStop struct {
	row  int
	file string
}

// mapStops is the map's files in the order they are drawn.
func (m Model) mapStops() []mapStop {
	files := panes.MapFiles(m.paneEnv(tabMap))

	out := make([]mapStop, 0, len(files))
	for row, file := range files {
		out = append(out, mapStop{row: row, file: file})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].row < out[j].row })

	return out
}

// stepMap moves the cursor by one file, round the ends. With nothing under
// it yet, down is the first file and up the last.
func (m Model) stepMap(by int) Model {
	stops := m.mapStops()
	if len(stops) == 0 {
		return m
	}

	next := 0
	if by < 0 {
		next = len(stops) - 1
	}

	for i, s := range stops {
		if s.file == m.mapAt {
			next = (i + by + len(stops)) % len(stops)

			break
		}
	}

	m.mapAt = stops[next].file

	return m.showMapRow(stops[next].row).syncPanes()
}

// showMapRow scrolls the pane far enough that the cursor is on screen, and
// no further, the way the flow tree's does.
func (m Model) showMapRow(row int) Model {
	vp := m.panes[tabMap]
	_, rows := m.paneBandFor(tabMap)

	switch top := vp.YOffset(); {
	case row < top:
		vp.SetYOffset(row)
	case rows > 0 && row >= top+rows:
		vp.SetYOffset(row - rows + 1)
	default:
		return m
	}

	m.panes[tabMap] = vp

	return m
}

// withMapCaret is the map with every row given the gutter the caret is
// drawn into, and the caret on the cursor's file. The same width either
// way, so no branch moves when the cursor arrives.
func (m Model) withMapCaret(rows []string) []string {
	at := -1

	for _, s := range m.mapStops() {
		if s.file == m.mapAt {
			at = s.row
		}
	}

	out := make([]string, len(rows))
	for i, r := range rows {
		if i == at {
			out[i] = theme.Paint(theme.Live).Render(treeCaret) + r

			continue
		}

		out[i] = treeGutter + r
	}

	return out
}
