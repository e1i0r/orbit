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
