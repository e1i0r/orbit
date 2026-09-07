package compose

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/typing"
)

func (s State) isComposeFlowField() bool {
	return (s.tab == composeTabManual && s.field == composeFlow) ||
		(s.tab == composeTabURL && s.field == composeURLFlow)
}

// isPillField is the flow row, which is the only row of the form that is
// chosen from rather than typed into.
func (s State) isPillField() bool {
	return s.isComposeFlowField()
}

// The left and right arrows are the pills on a row of pills and the caret
// everywhere else. Held with the option key they are a word at a time,
// which is what the rest of the machine does with them.
func (s State) handleComposeLeft(word bool) State {
	switch {
	case s.isComposeFlowField():
		return s.cycleComposeFlow(-1)
	case word:
		return s.composeCaret((*typing.Field).WordLeft)
	}

	return s.composeCaret(func(in *typing.Field) { in.MoveBy(-1) })
}

func (s State) handleComposeRight(word bool) State {
	switch {
	case s.isComposeFlowField():
		return s.cycleComposeFlow(1)
	case word:
		return s.composeCaret((*typing.Field).WordRight)
	}

	return s.composeCaret(func(in *typing.Field) { in.MoveBy(1) })
}

// The three keys below are one movement each, and what was held down with
// it. The option key makes a side arrow a word at a time; the shift key
// makes any of them a selection, by running the very same movement with the
// anchor left where it was.
func (s State) composeArrow(d int, mod tea.KeyMod) State {
	word := mod&tea.ModAlt != 0

	move := func(mm State) State {
		if d < 0 {
			return mm.handleComposeLeft(word)
		}

		return mm.handleComposeRight(word)
	}

	if mod&tea.ModShift != 0 {
		return s.composeExtend(move)
	}

	return move(s)
}

func (s State) composeVertical(d int, mod tea.KeyMod, e Env) State {
	if mod&tea.ModShift != 0 {
		return s.composeExtend(func(mm State) State { return mm.composeUp(d, e) })
	}

	return s.composeUp(d, e)
}

func (s State) composeJump(move func(*typing.Field), mod tea.KeyMod) State {
	if mod&tea.ModShift != 0 {
		return s.composeCaret(func(in *typing.Field) { in.Extend(move) })
	}

	return s.composeCaret(move)
}

func (s State) cycleComposeFlow(d int) State {
	if len(s.flows) == 0 {
		return s
	}

	n := len(s.flows)
	s.flowIdx = (s.flowIdx + d + n) % n

	return s
}

func (s *State) refreshFlows(src flow.Source) {
	listed := flow.List(src)

	var flows []string
	for _, f := range listed {
		flows = append(flows, f.Name)
	}

	if len(flows) == 0 {
		flows = flow.BuiltinNames()
	}

	s.flows = flows
	if s.flowIdx >= len(s.flows) {
		s.flowIdx = len(s.flows) - 1
	}
}

func (s State) chosenFlow() string {
	if len(s.flows) == 0 || s.flowIdx < 0 || s.flowIdx >= len(s.flows) {
		return flow.Default
	}

	return s.flows[s.flowIdx]
}

func (s *State) setFlow(name string) {
	for i, f := range s.flows {
		if f == name {
			s.flowIdx = i
			return
		}
	}
}
