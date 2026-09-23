package ui

// Walking the flow tree from the keyboard.
//
// The tree grew buttons and every one of them was a click. Elio, looking
// at a node he could see and could not reach: "el mouse no funciona aca,
// no puedo ir con las flechitas". Everything on this screen answers both
// hands or it answers neither.
//
// Only this pane. The other eleven are text a reader scrolls, and arrows
// that jumped between the landmarks of a timeline would be a reader losing
// their place. A tree is made of nodes, so on a tree the arrows walk nodes.

import (
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

// treeAt is where the cursor is standing, by the node rather than by the
// row it is drawn on.
//
// By the node, because opening one moves every row under it: a cursor kept
// as a row number would slide off whatever it was on the moment it was
// used. The zero value is the head of the first node, which is where a
// tree nobody has walked yet has its cursor: a tree that showed no cursor
// until an arrow was pressed would be a tree whose arrows do nothing the
// first time.
type treeAt struct {
	node   int
	button bool
}

// treeStop is one place the cursor can stand: the row it is drawn on, and
// what standing there means.
type treeStop struct {
	row int
	at  treeAt
}

// treeStops is everywhere the cursor can stand on the tree as it is drawn
// now, top to bottom.
//
// Both maps the tree already hands out, merged and sorted: the heads it
// folds by and the buttons it is pressed by. Nothing new is measured, so
// the cursor cannot land somewhere the mouse would not.
func (m Model) treeStops() []treeStop {
	if m.tab != tabFlow {
		return nil
	}

	return stopsOf(m.heads[tabFlow], m.flowButtons())
}

// stopsOf is the same list built from two maps in hand, which is what
// drawing the caret has: the tree is built once per sync and the cursor is
// marked on the rows that came out of that build.
func stopsOf(heads, buttons map[int]int) []treeStop {
	out := make([]treeStop, 0, len(heads)+len(buttons))
	for row, node := range heads {
		out = append(out, treeStop{row: row, at: treeAt{node: node}})
	}

	for row, node := range buttons {
		out = append(out, treeStop{row: row, at: treeAt{node: node, button: true}})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].row < out[j].row })

	return out
}

// treeRow is the drawn row the cursor is on, and whether it is on one.
//
// A cursor whose node is no longer a stop — the node folded, the run moved
// on and took its button away — is a cursor on nothing, which is what the
// caret not being drawn says.
func (m Model) treeRow() (int, bool) { return rowOfStop(m.treeStops(), m.tree) }

// rowOfStop is where one cursor stands in a list of stops.
func rowOfStop(stops []treeStop, at treeAt) (int, bool) {
	for _, s := range stops {
		if s.at == at {
			return s.row, true
		}
	}

	return 0, false
}

// stepTree moves the cursor one stop, off either end round to the other,
// and onto the first one when it is nowhere.
func (m Model) stepTree(by int) (tea.Model, tea.Cmd) {
	stops := m.treeStops()
	if len(stops) == 0 {
		return m, nil
	}

	next := 0
	if by < 0 {
		next = len(stops) - 1
	}

	for i, s := range stops {
		if s.at == m.tree {
			next = (i + by + len(stops)) % len(stops)

			break
		}
	}

	m.tree = stops[next].at

	return m.showTreeRow(stops[next].row), nil
}

// showTreeRow scrolls the pane far enough that the cursor is on screen,
// and no further: a tree that jumped to put the cursor in the middle would
// move the rows the reader is using to find their way.
func (m Model) showTreeRow(row int) Model {
	vp := m.panes[tabFlow]
	_, rows := m.paneBandFor(tabFlow)

	switch top := vp.YOffset(); {
	case row < top:
		vp.SetYOffset(row)
	case rows > 0 && row >= top+rows:
		vp.SetYOffset(row - rows + 1)
	default:
		return m
	}

	m.panes[tabFlow] = vp

	return m
}

// pressTree is ↵ on the cursor: a head opens or closes, a button is
// pressed.
func (m Model) pressTree() (tea.Model, tea.Cmd) {
	if _, on := m.treeRow(); !on {
		return m, nil
	}

	if m.tree.button {
		return m.runFromPhase(m.tree.node)
	}

	return m.openPaneRow(m.tree.node)
}

// treeCaret marks the row the cursor is on, and treeGutter is what it is
// drawn over: the two cells every row of the tree already spends on
// nothing. The same width, so no branch moves when the cursor arrives.
const (
	treeCaret  = "▌ "
	treeGutter = "  "
)

// withTreeCaret is the tree with the cursor drawn on it.
func (m Model) withTreeCaret(rows []string, heads, buttons map[int]int) []string {
	row, on := rowOfStop(stopsOf(heads, buttons), m.tree)
	if !on || row >= len(rows) {
		return rows
	}

	marked := make([]string, len(rows))
	copy(marked, rows)

	// Every row of the tree opens on the same gutter, which is what the
	// caret is drawn into. A row that does not is a row this does not
	// touch, rather than one it shifts sideways by two.
	if rest, cut := strings.CutPrefix(marked[row], treeGutter); cut {
		marked[row] = theme.Paint(theme.Live).Render(treeCaret) + rest
	}

	return marked
}

// treeSaid is what ↵ would do where the cursor is standing, for the bar.
//
// The word changes with the row because the gesture does: the same key
// opens a node and presses a button, and a bar that said "open" over a
// button would be describing the key it was pressed on last.
func (m Model) treeSaid() string {
	p := m.opts.Words
	if m.tree.button {
		return p.T("key.press_button", "press it")
	}

	if m.rowOpen(tabFlow, m.tree.node) {
		return p.T("key.shut_node", "close")
	}

	return p.T("key.open_node", "open")
}
