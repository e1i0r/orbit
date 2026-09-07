package engines

// The keyboard and the mouse, and what one row does when it is chosen.

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Key is one press. Which keyboard it goes to is the mode: the setup steps
// have their own, so has the filter being typed into.
func (s State) Key(msg tea.KeyPressMsg, e Env) (State, Out) {
	if s.showingSetup {
		if key.Matches(msg, e.Keys.Back) || key.Matches(msg, e.Keys.Quit) || key.Matches(msg, e.Keys.Open) {
			s.showingSetup = false
			return s, Out{}
		}

		return s, Out{}
	}

	if s.typing {
		return s.engineFilterKey(msg, e), Out{}
	}

	rows := s.collectEngineRows(e)

	idxs := selectableEngineIndices(rows)
	if len(idxs) == 0 {
		if key.Matches(msg, e.Keys.Back) || key.Matches(msg, e.Keys.Quit) {
			return s.leave()
		}

		return s, Out{}
	}

	switch {
	case key.Matches(msg, e.Keys.Back) || key.Matches(msg, e.Keys.Quit):
		chip := Chip(s.knobs, e)

		next, out := s.leave()
		out.Said = e.Words.T("engines.updated", "updated dials: {chip}", about("chip", chip))

		return next, out
	case key.Matches(msg, e.Keys.Up):
		s.sel--
		if s.sel < 0 {
			s.sel = len(idxs) - 1
		}

		return s.keepEngineRowSeen(e), Out{}
	case key.Matches(msg, e.Keys.Down):
		s.sel++
		if s.sel >= len(idxs) {
			s.sel = 0
		}

		return s.keepEngineRowSeen(e), Out{}
	case key.Matches(msg, e.Keys.Filter):
		s.typing = true
		return s, Out{}
	case key.Matches(msg, e.Keys.Sideways):
		return s.foldKnob(msg.String() != "left", e), Out{}
	case key.Matches(msg, e.Keys.Open), msg.Text == " ":
		return s.Choose(s.sel, e)
	}

	return s, Out{}
}

// leave closes the screen and says which one it was opened from.
func (s State) leave() (State, Out) {
	return State{knobs: s.knobs}, Out{Leave: true, Back: s.back}
}

// Choose takes the row at that place in the list, which is what ⏎ and a
// click both do. The number counts the rows a reader can stand on: the
// headings are not among them, and neither the keyboard nor the mouse can
// land on one.
func (s State) Choose(at int, e Env) (State, Out) {
	rows := s.collectEngineRows(e)

	idxs := selectableEngineIndices(rows)
	if at < 0 || at >= len(idxs) {
		return s, Out{}
	}

	s.sel = at

	return s.take(rows[idxs[at]], e)
}

// take is what one row does when it is chosen.
func (s State) take(row engineRow, e Env) (State, Out) {
	if row.disabled {
		s.showingSetup = true
		s.setupEngine = row.engine

		return s, Out{}
	}

	var set []Setting

	switch row.kind {
	case rowEngine:
		// An engine already in force has nothing left to choose, so ⏎ on
		// its name folds it instead: the reader is done with its models,
		// or wants them back.
		if s.knobs.Engine == row.engine {
			return s.foldKnob(!row.open, e), Out{}
		}

		s.knobs.Engine, s.knobs.Model = row.engine, ""
		s = s.foldEngine(row.engine, true)
		set = []Setting{{Name: "engine", Value: row.engine}}
	case rowModel:
		s.knobs.Engine, s.knobs.Model = row.engine, row.id
		set = []Setting{{Name: "model", Value: row.id}}
	case rowEffort:
		s.knobs.Effort = row.id
	case rowThinking:
		s.knobs.Thinking = row.id
	}

	// Choosing an engine opens its models under it and closes the one that
	// was open, so the list this was picked from is not the list it leaves
	// behind: where the view sits has to be asked again.
	return s.keepEngineRowSeen(e), Out{Set: set}
}

// Wheel moves the selection by rows. It stops at either end rather than
// wrapping as the arrows do: a notch that carried the reader from the last
// model back to the top would read as the list jumping under their hand.
func (s State) Wheel(d int, e Env) State {
	n := len(selectableEngineIndices(s.collectEngineRows(e)))
	if n == 0 {
		return s
	}

	s.sel = min(max(s.sel+d, 0), n-1)

	return s.keepEngineRowSeen(e)
}
