package engines

// The engines list folds: an engine's models show under its name or stay
// behind it.

import "maps"

// engineOpen says whether an engine's models are on the list.
//
// Everything starts shut, including the engine in force. Four engines and
// their catalogues are eighty-odd rows, and a screen that opened one of them
// by itself would open the longest list on the machine — opencode's
// sixty-five — in front of a reader who came here to change the effort. The
// names fit on one screen; what is behind them is asked for.
func (s State) engineOpen(name string) bool { return s.open[name] }

// foldEngine puts an engine's models on the list or takes them off it.
//
// The map is cloned rather than written in place, for the reason folds.go
// clones its own: a Model is copied by every method that returns one and a
// map is not, so a key written here would fold a section on windows this
// method never returned.
func (s State) foldEngine(name string, open bool) State {
	held := maps.Clone(s.open)
	if held == nil {
		held = map[string]bool{}
	}

	held[name] = open
	s.open = held

	return s
}

// foldKnob is ← and → on the list.
//
// A model row folds the engine it belongs to rather than nothing, so ←
// halfway down sixty-four models is the way back out of them: the key works
// on the section the cursor is in and not only on the line it is on.
func (s State) foldKnob(open bool, e Env) State {
	rows := s.collectEngineRows(e)

	idxs := selectableEngineIndices(rows)
	if s.sel < 0 || s.sel >= len(idxs) {
		return s
	}

	row := rows[idxs[s.sel]]
	if row.disabled || row.engine == "" || (row.kind != rowEngine && row.kind != rowModel) {
		return s
	}

	s = s.foldEngine(row.engine, open)

	// Closing takes the row the cursor was on off the list, so the cursor
	// goes to the name that closed over it — the one row of that section
	// still there to stand on.
	if !open {
		s = s.selectKnobEngine(row.engine, e)
	}

	return s.keepEngineRowSeen(e)
}

// selectKnobEngine puts the cursor on an engine's own row.
func (s State) selectKnobEngine(name string, e Env) State {
	rows := s.collectEngineRows(e)
	for i, idx := range selectableEngineIndices(rows) {
		if rows[idx].kind == rowEngine && rows[idx].engine == name {
			s.sel = i
			break
		}
	}

	return s
}
