package flow

// A flow file is text somebody else wrote, and every word of it is drawn.
//
// Somebody else includes an engine: the designer's "draft it" asks one for
// a whole flow and saves what comes back, and an engine writes coloured
// output for a living. A name or a prompt holding an escape sequence is an
// instruction to the terminal that draws it — see internal/tame — so a flow
// is tamed as it is decoded, before anything can draw it or write it back.
//
// A gate's command is tamed with the rest of it, though it is run and not
// only read. Three screens draw a command and one shell runs it, and a
// command carrying a raw escape is not something anybody typed on purpose:
// a shell that needs a tab or an escape has its own way of writing one —
// $'\t', printf — and those are ordinary characters that survive this.
// The alternative is a command drawn in three places that each have to
// remember to tame it, and the one that forgets is the hole.

import "github.com/e1i0r/orbit/internal/tame"

// tamed is this flow with the control characters out of every field that is
// read as words.
func (f Flow) tamed() Flow {
	f.Name = tame.Text(f.Name)
	f.Description = tame.Text(f.Description)

	f.Phases = tamedPhases(f.Phases)

	return f
}

// tamedPhases is the same for a list of them, and for the list inside a
// loop: a loop's phases are drawn exactly as the flow's own are.
func tamedPhases(phases []Phase) []Phase {
	out := make([]Phase, 0, len(phases))

	for _, p := range phases {
		p.Name = tame.Text(p.Name)
		p.Engine = tame.Text(p.Engine)
		p.Model = tame.Text(p.Model)
		p.Effort = tame.Text(p.Effort)
		p.Thinking = tame.Text(p.Thinking)
		p.Prompt = tame.Text(p.Prompt)
		p.Gates = tamedGates(p.Gates)

		if p.Loop != nil {
			loop := *p.Loop
			loop.Phases = tamedPhases(loop.Phases)
			loop.Until = tamedGates(loop.Until)
			p.Loop = &loop
		}

		out = append(out, p)
	}

	if len(out) == 0 {
		return phases
	}

	return out
}

// tamedGates is a gate's name, the rule it came from, and the command it
// runs.
func tamedGates(gates []Gate) []Gate {
	out := make([]Gate, 0, len(gates))

	for _, g := range gates {
		g.Name = tame.Text(g.Name)
		g.Rule = tame.Text(g.Rule)
		g.Command = tame.Text(g.Command)

		out = append(out, g)
	}

	if len(out) == 0 {
		return gates
	}

	return out
}
