package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

func (m Model) isComposeFlowField() bool {
	return (m.compose.tab == composeTabManual && m.compose.field == composeFlow) ||
		(m.compose.tab == composeTabURL && m.compose.field == composeURLFlow)
}

// isPillField is the flow row, which is the only row of the form that is
// chosen from rather than typed into.
func (m Model) isPillField() bool {
	return m.isComposeFlowField()
}

// The left and right arrows are the pills on a row of pills and the caret
// everywhere else. Held with the option key they are a word at a time,
// which is what the rest of the machine does with them.
func (m Model) handleComposeLeft(word bool) Model {
	switch {
	case m.isComposeFlowField():
		return m.cycleComposeFlow(-1)
	case word:
		return m.composeCaret((*input).wordLeft)
	}

	return m.composeCaret(func(in *input) { in.moveBy(-1) })
}

func (m Model) handleComposeRight(word bool) Model {
	switch {
	case m.isComposeFlowField():
		return m.cycleComposeFlow(1)
	case word:
		return m.composeCaret((*input).wordRight)
	}

	return m.composeCaret(func(in *input) { in.moveBy(1) })
}

// The three keys below are one movement each, and what was held down with
// it. The option key makes a side arrow a word at a time; the shift key
// makes any of them a selection, by running the very same movement with the
// anchor left where it was.
func (m Model) composeArrow(d int, mod tea.KeyMod) Model {
	word := mod&tea.ModAlt != 0

	move := func(mm Model) Model {
		if d < 0 {
			return mm.handleComposeLeft(word)
		}

		return mm.handleComposeRight(word)
	}

	if mod&tea.ModShift != 0 {
		return m.composeExtend(move)
	}

	return move(m)
}

func (m Model) composeVertical(d int, mod tea.KeyMod) Model {
	if mod&tea.ModShift != 0 {
		return m.composeExtend(func(mm Model) Model { return mm.composeUp(d) })
	}

	return m.composeUp(d)
}

func (m Model) composeJump(move func(*input), mod tea.KeyMod) Model {
	if mod&tea.ModShift != 0 {
		return m.composeCaret(func(in *input) { in.extend(move) })
	}

	return m.composeCaret(move)
}

func (m Model) cycleComposeFlow(d int) Model {
	if len(m.compose.flows) == 0 {
		return m
	}

	n := len(m.compose.flows)
	m.compose.flowIdx = (m.compose.flowIdx + d + n) % n

	return m
}

func (c *composeState) refreshFlows(src flow.Source) {
	listed := flow.List(src)

	var flows []string
	for _, f := range listed {
		flows = append(flows, f.Name)
	}

	if len(flows) == 0 {
		flows = flow.BuiltinNames()
	}

	c.flows = flows
	if c.flowIdx >= len(c.flows) {
		c.flowIdx = len(c.flows) - 1
	}
}

func (c composeState) chosenFlow() string {
	if len(c.flows) == 0 || c.flowIdx < 0 || c.flowIdx >= len(c.flows) {
		return flow.Default
	}

	return c.flows[c.flowIdx]
}

func (c *composeState) setFlow(name string) {
	for i, f := range c.flows {
		if f == name {
			c.flowIdx = i
			return
		}
	}
}
