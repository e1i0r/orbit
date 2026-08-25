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
	return m.isComposeRepoField() || m.isComposeFlowField() || m.isComposeModelField() ||
		m.isComposeThinkingField() || m.isComposeEffortField()
}

func (m Model) handleComposeLeft() Model {
	switch {
	case m.isComposeRepoField():
		return m.cycleComposeRepo(-1)
	case m.isComposeFlowField():
		return m.cycleComposeFlow(-1)
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
		flows = []string{"task", "quick", "careful"}
	}
	c.flows = flows
	if c.flowIdx >= len(c.flows) {
		c.flowIdx = len(c.flows) - 1
	}
}

func (c composeState) chosenFlow() string {
	if len(c.flows) == 0 || c.flowIdx < 0 || c.flowIdx >= len(c.flows) {
		return "task"
	}
	return c.flows[c.flowIdx]
}

func (c composeState) chosenModel() string {
	if len(c.models) == 0 || c.modelIdx < 0 || c.modelIdx >= len(c.models) {
		return "sonnet"
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
		return "default"
	}
	return c.efforts[c.effortIdx]
}
