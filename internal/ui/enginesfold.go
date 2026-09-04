package ui

// The engines list folds: an engine's models show under its name or stay
// behind it.

import "maps"

// engineOpen says whether an engine's models are on the list.
//
// Absent from the map is the engine in force open and every other one shut,
// which is this screen as it was before it folded at all: a reader who has
// touched no arrow sees the models of the engine the run will use, and the
// rest of the engines by name.
func (m Model) engineOpen(name string) bool {
	if shown, set := m.engines.open[name]; set {
		return shown
	}

	return name == m.dialEngine(m.knobs.Engine)
}

// foldEngine puts an engine's models on the list or takes them off it.
//
// The map is cloned rather than written in place, for the reason folds.go
// clones its own: a Model is copied by every method that returns one and a
// map is not, so a key written here would fold a section on windows this
// method never returned.
func (m Model) foldEngine(name string, open bool) Model {
	held := maps.Clone(m.engines.open)
	if held == nil {
		held = map[string]bool{}
	}

	held[name] = open
	m.engines.open = held

	return m
}

// foldKnob is ← and → on the list.
//
// A model row folds the engine it belongs to rather than nothing, so ←
// halfway down sixty-four models is the way back out of them: the key works
// on the section the cursor is in and not only on the line it is on.
func (m Model) foldKnob(open bool) Model {
	rows := m.collectEngineRows()

	idxs := m.selectableEngineIndices(rows)
	if m.engines.sel < 0 || m.engines.sel >= len(idxs) {
		return m
	}

	row := rows[idxs[m.engines.sel]]
	if row.disabled || row.engine == "" || (row.kind != rowEngine && row.kind != rowModel) {
		return m
	}

	m = m.foldEngine(row.engine, open)

	// Closing takes the row the cursor was on off the list, so the cursor
	// goes to the name that closed over it — the one row of that section
	// still there to stand on.
	if !open {
		m = m.selectKnobEngine(row.engine)
	}

	return m.keepEngineRowSeen()
}

// selectKnobEngine puts the cursor on an engine's own row.
func (m Model) selectKnobEngine(name string) Model {
	rows := m.collectEngineRows()
	for i, idx := range m.selectableEngineIndices(rows) {
		if rows[idx].kind == rowEngine && rows[idx].engine == name {
			m.engines.sel = i
			break
		}
	}

	return m
}
