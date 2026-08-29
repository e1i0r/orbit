package ui

import (
	"github.com/e1i0r/orbit/internal/flow"
)

func (m Model) isComposeRepoField() bool {
	return (m.compose.tab == composeTabManual && m.compose.field == composeRepo) ||
		(m.compose.tab == composeTabURL && m.compose.field == composeURLRepo)
}

func (m Model) isComposeFlowField() bool {
	return (m.compose.tab == composeTabManual && m.compose.field == composeFlow) ||
		(m.compose.tab == composeTabURL && m.compose.field == composeURLFlow)
}

func (m Model) isComposeEngineField() bool {
	return (m.compose.tab == composeTabManual && m.compose.field == composeEngine) ||
		(m.compose.tab == composeTabURL && m.compose.field == composeURLEngine)
}

func (m Model) isComposeModelField() bool {
	return (m.compose.tab == composeTabManual && m.compose.field == composeModel) ||
		(m.compose.tab == composeTabURL && m.compose.field == composeURLModel)
}

func (m Model) isComposeThinkingField() bool {
	return (m.compose.tab == composeTabManual && m.compose.field == composeThinking) ||
		(m.compose.tab == composeTabURL && m.compose.field == composeURLThinking)
}

func (m Model) isComposeEffortField() bool {
	return (m.compose.tab == composeTabManual && m.compose.field == composeEffort) ||
		(m.compose.tab == composeTabURL && m.compose.field == composeURLEffort)
}

func (m Model) isPillField() bool {
	return m.isComposeRepoField() || m.isComposeFlowField() || m.isComposeEngineField() ||
		m.isComposeModelField() || m.isComposeThinkingField() || m.isComposeEffortField()
}

func (m Model) handleComposeLeft() Model {
	switch {
	case m.isComposeRepoField():
		return m.cycleComposeRepo(-1)
	case m.isComposeFlowField():
		return m.cycleComposeFlow(-1)
	case m.isComposeEngineField():
		return m.cycleComposeEngine(-1)
	case m.isComposeModelField():
		return m.cycleComposeModel(-1)
	case m.isComposeThinkingField():
		return m.cycleComposeThinking(-1)
	case m.isComposeEffortField():
		return m.cycleComposeEffort(-1)
	}

	return m
}

func (m Model) handleComposeRight() Model {
	switch {
	case m.isComposeRepoField():
		return m.cycleComposeRepo(1)
	case m.isComposeFlowField():
		return m.cycleComposeFlow(1)
	case m.isComposeEngineField():
		return m.cycleComposeEngine(1)
	case m.isComposeModelField():
		return m.cycleComposeModel(1)
	case m.isComposeThinkingField():
		return m.cycleComposeThinking(1)
	case m.isComposeEffortField():
		return m.cycleComposeEffort(1)
	}

	return m
}

func (m Model) cycleComposeRepo(d int) Model {
	if len(m.compose.repos) == 0 {
		return m
	}

	n := len(m.compose.repos)
	m.compose.repoIdx = (m.compose.repoIdx + d + n) % n
	m.compose.repo = m.compose.repos[m.compose.repoIdx].name

	return m
}

func (m Model) cycleComposeFlow(d int) Model {
	if len(m.compose.flows) == 0 {
		return m
	}

	n := len(m.compose.flows)
	m.compose.flowIdx = (m.compose.flowIdx + d + n) % n

	return m
}

func (m Model) cycleComposeEngine(d int) Model {
	n := len(m.compose.engines)
	if n == 0 {
		return m
	}

	return m.chooseComposeEngine((m.compose.engineIdx + d + n) % n)
}

// chooseComposeEngine moves the engine dial and refills the two dials that
// hang off it.
//
// Three gestures land here — opening the form, the arrow keys, a click on a
// pill — and the first two of them used to carry their own copy of this,
// which is why a click could leave the model dial showing another engine's
// models.
func (m Model) chooseComposeEngine(i int) Model {
	if i < 0 || i >= len(m.compose.engines) {
		return m
	}

	m.compose.engineIdx = i
	eng := m.compose.engines[i]

	m.compose.models, m.compose.modelLabels = m.modelsFor(eng)
	if m.compose.modelIdx >= len(m.compose.models) {
		m.compose.modelIdx = 0
	}

	m.compose.efforts, m.compose.effortLabels = m.effortsFor(eng)
	if m.compose.effortIdx >= len(m.compose.efforts) {
		m.compose.effortIdx = 0
	}

	return m
}

func (m Model) cycleComposeModel(d int) Model {
	if len(m.compose.models) == 0 {
		return m
	}

	n := len(m.compose.models)
	m.compose.modelIdx = (m.compose.modelIdx + d + n) % n

	return m
}

func (m Model) cycleComposeThinking(d int) Model {
	if len(m.compose.thinkings) == 0 {
		return m
	}

	n := len(m.compose.thinkings)
	m.compose.thinkingIdx = (m.compose.thinkingIdx + d + n) % n

	return m
}

func (m Model) cycleComposeEffort(d int) Model {
	if len(m.compose.efforts) == 0 {
		return m
	}

	n := len(m.compose.efforts)
	m.compose.effortIdx = (m.compose.effortIdx + d + n) % n

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

func (c composeState) chosenEngine() string {
	if len(c.engines) == 0 || c.engineIdx < 0 || c.engineIdx >= len(c.engines) {
		return ""
	}

	return c.engines[c.engineIdx]
}

func (c composeState) chosenModel() string {
	if len(c.models) == 0 || c.modelIdx < 0 || c.modelIdx >= len(c.models) {
		return ""
	}

	return c.models[c.modelIdx]
}

func (c composeState) chosenThinking() string {
	if len(c.thinkings) == 0 || c.thinkingIdx < 0 || c.thinkingIdx >= len(c.thinkings) {
		return "adaptive"
	}

	return c.thinkings[c.thinkingIdx]
}

func (c composeState) chosenEffort() string {
	if len(c.efforts) == 0 || c.effortIdx < 0 || c.effortIdx >= len(c.efforts) {
		return ""
	}

	return c.efforts[c.effortIdx]
}

func (c *composeState) setFlow(name string) {
	for i, f := range c.flows {
		if f == name {
			c.flowIdx = i
			return
		}
	}
}
