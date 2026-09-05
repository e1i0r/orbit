package ui

// Building a loop in the designer, rather than by writing the JSON by hand.
//
// A loop is not a fifth kind of thing on this screen: it is a phase with the
// repeat switch on. Everything already typed into that phase — the engine,
// the model, the prompt — goes on describing what runs, and two fields
// appear beside it: how many turns it may take, and what has to pass for it
// to stop. That is the whole of the shape, and it is why toggling the switch
// never asks the reader to type anything again.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
)

// defaultTurns is what a loop is born with. Three is the number the built-in
// flows use: enough for a fix to land after a check has told it twice what is
// wrong, and short enough that a wall costs three runs and not a quota window.
const defaultTurns = 3

// edited is the phase the engine, model and prompt fields describe.
//
// For a loop that is the phase inside it, because a loop runs nothing of its
// own — internal/flow refuses a phase that is both an engine and a loop. For
// everything else it is the phase itself.
func (st *flowsState) edited() *flow.Phase {
	ph := st.cur()
	if ph.Loop != nil && len(ph.Loop.Phases) > 0 {
		return &ph.Loop.Phases[0]
	}

	return ph
}

// looping is whether the phase being edited repeats.
func (st *flowsState) looping() bool {
	return st.cur().Loop != nil
}

// toggleLoop turns the phase under the cursor into a loop, or back.
//
// What was already written is what goes round: somebody who has typed a
// prompt and then decides it should repeat until the tests pass has not
// changed their mind about the prompt. The outer phase keeps the name and
// the human stop, because those are about the step in the pipeline; the rest
// moves inside, where the engine that runs it lives.
func (m Model) toggleLoop() Model {
	st := &m.flows
	ph := st.cur()

	if ph.Loop != nil {
		inner := ph.Loop.Phases[0]
		inner.Name, inner.Wait, inner.Loop = ph.Name, ph.Wait, nil
		*ph = inner

		return m
	}

	inner := *ph
	inner.Wait = false

	*ph = flow.Phase{
		Name: ph.Name,
		Wait: ph.Wait,
		Loop: &flow.Loop{Phases: []flow.Phase{inner}, Max: defaultTurns},
	}

	return m
}

// setLoopTurns says how much rope the loop gets, and never none: a loop
// capped at zero turns is a phase that cannot run, which is not something
// anybody means by editing this field.
func (m Model) setLoopTurns(n int) Model {
	ph := m.flows.cur()
	if ph.Loop == nil {
		return m
	}

	ph.Loop.Max = max(n, 1)

	return m
}

// setLoopChecks reads the checks out of the field they are typed into: one
// per line, a name, a colon, and the command whose exit code answers.
//
// What was typed is kept beside what was parsed, because the two are not the
// same string: half a line — "tests" with the colon not yet reached — parses
// into a check named for its position, and a field redrawn from the parse
// would rewrite itself under the reader's hands mid-word.
//
// A line with no colon is kept rather than dropped, under a name of its own.
// Losing a typed line silently is worse than a check called "check 2", and
// internal/flow refuses a check with no name at all.
func (m Model) setLoopChecks(text string) Model {
	m.flows.setChecks(text)

	return m
}

// setChecks is the same thing on the state alone, which is where the keys
// that type into the field reach it from.
func (st *flowsState) setChecks(text string) {
	ph := st.cur()
	if ph.Loop == nil {
		return
	}

	st.checksDraft, st.checksTyped = text, true
	st.checksFor = st.activePhase

	var until []flow.Gate

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		name, command, held := strings.Cut(line, ":")
		if !held || strings.TrimSpace(command) == "" {
			name, command = fmt.Sprintf("check %d", len(until)+1), line
		}

		until = append(until, flow.Gate{
			Name:    strings.TrimSpace(name),
			Command: strings.TrimSpace(command),
		})
	}

	ph.Loop.Until = until
}

// loopChecksText is what the checks field holds: what was typed into it if
// this is the phase it was typed for, and otherwise the checks the phase
// carries, written out the way they are read back in.
func (st *flowsState) loopChecksText() string {
	ph := st.cur()
	if ph.Loop == nil {
		return ""
	}

	if st.checksTyped && st.checksFor == st.activePhase {
		return st.checksDraft
	}

	lines := make([]string, 0, len(ph.Loop.Until))
	for _, g := range ph.Loop.Until {
		lines = append(lines, g.Name+": "+g.Command)
	}

	return strings.Join(lines, "\n")
}

// loopTurnsText is the cap as the field shows it.
func (st *flowsState) loopTurnsText() string {
	ph := st.cur()
	if ph.Loop == nil {
		return ""
	}

	return strconv.Itoa(ph.Loop.Max)
}
