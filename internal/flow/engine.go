package flow

// Running a flow with an engine other than the one it names.
//
// A flow says which engine walks each phase, and that is right: a flow is a
// shape of work, and "review with the careful model" is part of the shape.
// What it cannot know is that the engine it names has run out of quota at
// four in the afternoon, and that the person watching would rather carry on
// with another one than wait three hours.
//
// So the override is on the run and not on the flow. The file on disk is
// untouched; what changes is the copy this run walks, and the record says
// which engine each phase actually used because phase.started carries it.

// WithEngine is this flow with every phase pointed at one engine.
//
// A copy, and a deep one: a loop holds its phases behind a pointer, and a
// shallow copy would reach through it and rewrite the flow every other run
// of Orbit is about to read off disk.
//
// The model is cleared with it. A model is a name one engine knows —
// "opus" means nothing to codex — so carrying it across would ask the new
// engine for a dial it does not have. The engine's own default is the only
// honest answer, and it is what a phase that names no model already gets.
func WithEngine(f Flow, name string) Flow {
	if name == "" {
		return f
	}

	out := f
	out.Phases = make([]Phase, 0, len(f.Phases))

	for _, p := range f.Phases {
		out.Phases = append(out.Phases, phaseWith(p, name))
	}

	return out
}

// phaseWith is one phase pointed at an engine, and the block under it.
func phaseWith(p Phase, name string) Phase {
	out := p
	out.Engine, out.Model = name, ""

	if p.Loop == nil {
		return out
	}

	loop := *p.Loop
	loop.Phases = make([]Phase, 0, len(p.Loop.Phases))

	for _, inner := range p.Loop.Phases {
		loop.Phases = append(loop.Phases, phaseWith(inner, name))
	}

	out.Loop = &loop

	return out
}
