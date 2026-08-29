package ui

import (
	"maps"

	tea "charm.land/bubbletea/v2"
)

// Folding: the sections of the task view open and close, and the head of
// each one says which it is.
//
// The overview holds five blocks and a reader looking for one of them reads
// past the other four. Folding is what a reader does to a screen they come
// back to — the phases matter while a run is going and the deliver keys
// matter when it is over — so the state is the window's and not the task's.

// The sections that fold. The keys are this package's own vocabulary rather
// than the labels above them, which are translated and would change what a
// fold applies to when the window's language changed.
const (
	foldPhases  = "phases"
	foldChanges = "changes"
	foldDeliver = "deliver"
)

// overviewFolds is every section of the overview, in the order the pane
// draws them.
var overviewFolds = []string{foldPhases, foldChanges, foldDeliver}

// folded says whether a section is closed. Absent is open, so a window
// nobody has folded anything in shows everything it has.
func (m Model) folded(key string) bool { return m.folds[key] }

// sectionHead is one section's head, in the state that section is in.
func (m Model) sectionHead(key, label, note string, w int) string {
	return section(label, note, w, !m.folded(key))
}

// fold closes an open section and opens a closed one.
//
// The map is cloned rather than written in place: a Model is copied by every
// method that returns one, and a map is not — writing a key here would fold
// a section on windows this method never returned.
func (m Model) fold(key string) Model {
	held := maps.Clone(m.folds)
	if held == nil {
		held = map[string]bool{}
	}

	if held[key] {
		delete(held, key)
	} else {
		held[key] = true
	}

	m.folds = held

	return m.syncPanes()
}

// rowOpen says whether one row of a pane shows everything it has: the reader
// opened it, or [e] opened every row at once.
//
// It takes the tab because a pane is drawn whether or not it is the one on
// screen — syncPanes lays out all eleven — so the pane being asked about is
// never assumed to be the pane in front of the reader.
func (m Model) rowOpen(t tab, i int) bool { return m.expandedDetail || m.opened[t][i] }

// openRow opens a closed row of the pane that is up and closes an open one.
//
// The diff is the one pane that keeps this by path rather than by row, so it
// is sent to its own toggle: a file is the same file after a rebuild has
// moved every row under it.
func (m Model) openRow(i int) Model {
	if m.tab == tabDiff {
		return m.collapseFileAt(i)
	}

	held := maps.Clone(m.opened[m.tab])
	if held == nil {
		held = map[int]bool{}
	}

	if held[i] {
		delete(held, i)
	} else {
		held[i] = true
	}

	m.opened[m.tab] = held

	return m.syncPanes()
}

// openPaneRow opens a row of the pane that is up and asks for whatever that
// row needs before it can be shown.
//
// The artifacts tab is the one pane whose rows are not already in hand: a
// file is read when it is opened and not before, because reading every file
// of the directory on every tick to draw a list of names would be a read
// nobody asked for. It is asked for once — a second open shows what the
// first one brought back.
func (m Model) openPaneRow(i int) (Model, tea.Cmd) {
	m = m.openRow(i)
	if m.tab != tabArtifacts || !m.rowOpen(tabArtifacts, i) || i < 0 || i >= len(m.files) {
		return m, nil
	}

	name := m.files[i].Name
	if _, got := m.read[name]; got {
		return m, nil
	}

	return m, fileTextOf(m.opts.Reader, m.subject(), name)
}

// attemptOpen says whether an attempt shows the phases under it.
func (m Model) attemptOpen(n int) bool { return !m.shutAttempts[n] }

// foldAttempt closes an open attempt and opens a closed one.
//
// An attempt is the whole of one run, so this is the coarsest fold the task
// view has: a task tried three times holds three reports, and two of them
// are history the moment the third one starts.
func (m Model) foldAttempt(n int) Model {
	held := maps.Clone(m.shutAttempts)
	if held == nil {
		held = map[int]bool{}
	}

	if held[n] {
		delete(held, n)
	} else {
		held[n] = true
	}

	m.shutAttempts = held

	return m.syncPanes()
}

// hitSeam is the attempt, if any, whose rule was drawn on a row of the pane
// that is up.
//
// It asks no question about which tab that is: a pane that draws no seams
// has no map, and a nil map answers no for every row.
func (m Model) hitSeam(row int) (int, bool) {
	vp := m.panes[m.tab]

	n, ok := m.seams[m.tab][row+vp.YOffset()]

	return n, ok
}

// hitPaneRow is the entry, if any, whose own row was drawn on a row of the
// pane that is up, counting from the first row the pane drew and through
// whatever the reader has scrolled past.
func (m Model) hitPaneRow(row int) (int, bool) {
	vp := m.panes[m.tab]

	i, ok := m.heads[m.tab][row+vp.YOffset()]

	return i, ok
}

// foldAll closes every section of the overview, or opens every one of them
// when they are already closed. It is the one gesture that cleans the screen
// down to its heads without a pointer.
func (m Model) foldAll() Model {
	shut := map[string]bool{}
	for _, key := range overviewFolds {
		shut[key] = true
	}

	if maps.Equal(m.folds, shut) {
		m.folds = nil

		return m.syncPanes()
	}

	m.folds = shut

	return m.syncPanes()
}

// overviewFoldRows is which row of the pane's content each section head
// landed on.
//
// A head is the first row of its block, so where it landed is the count of
// what the pane drew before it. The blocks are asked their own heights here
// rather than searched for in the drawn rows, which keeps a translated label
// out of a hit test — and pins this arithmetic to overviewLines, which is
// what the test beside it is for.
func (m Model) overviewFoldRows() map[int]string {
	t, ok := m.task(m.detail)
	if !ok || m.logErr != nil {
		return nil
	}

	w := max(40, m.frame.Body.W)

	// The blank line overviewLines opens the pane with.
	at := 1 + len(m.overviewHead(t, w)) + len(m.overviewVitals(t, w))

	rows := map[int]string{at: foldPhases}

	at += len(m.overviewPhases(t, w))
	rows[at] = foldChanges

	at += len(m.overviewChanges(w))
	rows[at] = foldDeliver

	return rows
}

// hitFold is the section head, if any, on a row of the overview pane,
// counting from the first row the pane drew and through whatever the reader
// has scrolled past.
func (m Model) hitFold(row int) (string, bool) {
	if m.tab != tabOverview {
		return "", false
	}

	vp := m.panes[tabOverview]

	key, ok := m.overviewFoldRows()[row+vp.YOffset()]

	return key, ok
}
