package ui

import "github.com/e1i0r/orbit/internal/ui/roster"

import "github.com/e1i0r/orbit/internal/ui/cells"

// The one answer in this package to "what engines are there, and what do
// they offer".
//
// One rather than four. A claude table on the engines screen to fall back
// on, a second in settingsdials.go, a third in compose.go, and the real one
// — the adapters in internal/engine, handed over through Options.Engines —
// read by none of them: the copies agree neither with each other nor with
// the engines, offering opencode a model called llama-3.3 on one dial and
// gemini-2.5-pro on another, neither of which opencode has ever had.
//
// There is no fabricated table here. A window whose Engines port is nil has
// nothing to say about engines, and saying nothing is the only honest answer
// available: this package may not name internal/engine, so anything it made
// up would be a fourth copy waiting to drift.

// engineTable is every engine the build can run, as the port answers.
func (m Model) engineTable() []roster.Engine {
	if m.opts.Engines == nil {
		return nil
	}

	return m.opts.Engines()
}

// engineNames is the engines a dial can offer, in the order the port gave
// them — which internal/cli sorts, so the rows do not move between two
// openings of the same screen.
func (m Model) engineNames() []string {
	table := m.engineTable()

	names := make([]string, 0, len(table))
	for _, eng := range table {
		names = append(names, eng.Name)
	}

	return names
}

// modelsFor and effortsFor are one engine's dials: the ids a setting holds,
// and the labels a reader picks from.
//
// They are two lists and not one because they are two different strings for
// opencode — the id is provider-qualified, opencode/claude-opus-5, and what
// belongs on a dial is claude-opus-5. Drawing the id would put the provider
// in front of every position; saving the label would save a model opencode
// does not answer to.
func (m Model) modelsFor(engine string) (ids, labels []string) {
	info, ok := m.engineInfo(engine)
	if !ok {
		return nil, nil
	}

	return dialOf(info.Models)
}

func (m Model) effortsFor(engine string) (ids, labels []string) {
	info, ok := m.engineInfo(engine)
	if !ok {
		return nil, nil
	}

	return dialOf(info.Efforts)
}

// engineInfo is one engine by name, and whether it was there at all.
func (m Model) engineInfo(name string) (roster.Engine, bool) {
	for _, eng := range m.engineTable() {
		if eng.Name == name {
			return eng, true
		}
	}

	return roster.Engine{}, false
}

// dialOf splits a port's choices into what is stored and what is drawn, and
// leaves out the choice whose id is empty.
//
// That one means "whatever the engine is configured for", which is a real
// answer on the engines screen and not one this dial can carry: a click is
// routed back by the value it drew, and an empty value is indistinguishable
// from a click on no pill at all.
func dialOf(choices []roster.Choice) (ids, labels []string) {
	for _, c := range choices {
		if c.ID == "" {
			continue
		}

		label := c.Label
		if label == "" {
			label = c.ID
		}

		ids = append(ids, c.ID)
		labels = append(labels, label)
	}

	return ids, labels
}

// dialEngine is whose dials to offer when nothing names an engine: the
// window's setting, and failing that the first engine this build has.
//
// A phase inherits its engine, and so does the start dialog, so "" is the
// ordinary case here and not a fault. What it must not become is a name
// picked out of the air: answering claude to it names an engine that may
// not even be installed on this build.
func (m Model) dialEngine(named string) string {
	if named != "" {
		return named
	}

	if s := m.opts.Settings; s != nil {
		if set := s.Engine(); set != "" {
			return set
		}
	}

	return cells.First(m.engineNames())
}
