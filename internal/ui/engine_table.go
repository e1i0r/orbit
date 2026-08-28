package ui

// The one answer in this package to "what engines are there, and what do
// they offer".
//
// It used to be four answers. The engines screen carried a claude table to
// fall back on, the settings dials carried a second in settingsdials.go, the
// compose dials carried a third in compose.go, and the real one — the
// adapters in internal/engine, handed over through Options.Engines — was
// read by none of them. The three copies did not agree with each other or
// with the engines: the compose dial offered opencode a model called
// llama-3.3 and the settings dial offered it gemini-2.5-pro, neither of
// which opencode has ever had.
//
// There is no fabricated table here. A window whose Engines port is nil has
// nothing to say about engines, and saying nothing is the only honest answer
// available: this package may not name internal/engine, so anything it made
// up would be a fourth copy waiting to drift.

// engineTable is every engine the build can run, as the port answers.
func (m Model) engineTable() []EngineInfo {
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
func (m Model) engineInfo(name string) (EngineInfo, bool) {
	for _, eng := range m.engineTable() {
		if eng.Name == name {
			return eng, true
		}
	}

	return EngineInfo{}, false
}

// dialOf splits a port's choices into what is stored and what is drawn, and
// leaves out the choice whose id is empty.
//
// That one means "whatever the engine is configured for", which is a real
// answer on the engines screen and not one this dial can carry: a click is
// routed back by the value it drew, and an empty value is indistinguishable
// from a click on no pill at all.
func dialOf(choices []ChoiceInfo) (ids, labels []string) {
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

// dialLabel is what the option at i on a dial is drawn as: the label beside
// it when there is one, and the option itself when there is not.
//
// It takes two slices rather than a slice of pairs because the ids are what
// every dial in this package already holds, compares and stores, and the
// labels are only ever read at the moment of drawing.
func dialLabel(ids, labels []string, i int) string {
	if i < 0 || i >= len(ids) {
		return ""
	}

	if i < len(labels) {
		return labels[i]
	}

	return ids[i]
}

// dialEngine is whose dials to offer when nothing names an engine: the
// window's setting, and failing that the first engine this build has.
//
// A phase inherits its engine, and so does the start dialog, so "" is the
// ordinary case here and not a fault. What it must not become is a name
// picked out of the air — the dials in this package used to answer claude
// to it, on a build where claude may not even be installed.
func (m Model) dialEngine(named string) string {
	if named != "" {
		return named
	}

	if s := m.opts.Settings; s != nil {
		if set := s.Engine(); set != "" {
			return set
		}
	}

	return first(m.engineNames())
}
