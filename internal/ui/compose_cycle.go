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

func (m Model) isPillField() bool {
	return m.isComposeRepoField() || m.isComposeFlowField()
}

func (m Model) handleComposeLeft() Model {
	switch {
	case m.isComposeRepoField():
		return m.cycleComposeRepo(-1)
	case m.isComposeFlowField():
		return m.cycleComposeFlow(-1)
	}

	return m
}

func (m Model) handleComposeRight() Model {
	switch {
	case m.isComposeRepoField():
		return m.cycleComposeRepo(1)
	case m.isComposeFlowField():
		return m.cycleComposeFlow(1)
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
