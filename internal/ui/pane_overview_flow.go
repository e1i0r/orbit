package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
)

// finishedPhases is every phase of this task that stopped, however it
// stopped.
func (m Model) finishedPhases() []view.Entry {
	var done []view.Entry

	for _, e := range m.entries {
		if e.Phase == "" {
			continue
		}

		switch e.What() {
		case view.EntryFinished, view.EntryFailed, view.EntryCancelled:
			done = append(done, e)
		}
	}

	return done
}

// overviewPhases renders the single focused Flow card in Overview: what the
// flow is doing right now (or how it concluded), while leaving the complete
// multi-phase historical tree and details for Tab 2 [Flow].
func (m Model) overviewPhases(t view.Task, w int) []string {
	p := m.opts.Words
	flowName := orDef(t.Flow, flow.Default)
	head := m.sectionHead(foldPhases, p.T("overview.execution_summary", "flow"), flowName, w)

	if m.folded(foldPhases) {
		return []string{head, ""}
	}

	out := []string{head}

	switch t.Band {
	case view.Running:
		out = append(out, m.liveFlowCard(t, flowName, w)...)
	case view.NeedsYou:
		out = append(out, m.waitingFlowCard(t, flowName)...)
	case view.Done:
		out = append(out, m.doneFlowCard(t, flowName)...)
	default:
		out = append(out, m.todoFlowCard(flowName)...)
	}

	return append(out, "")
}

func (m Model) liveFlowCard(t view.Task, flowName string, w int) []string {
	p := m.opts.Words
	glyph := m.runGlyph(working(t))
	step := orDef(t.Phase, "running")
	now := orDef(t.CurrentAction, p.T("overview.running_model", "running model..."))

	now = fit(now, max(20, w-lipgloss.Width(paneGutter)-lipgloss.Width(glyph)-4))

	out := []string{
		paneGutter + Paint(Live).Render(glyph) + Text(Primary).Bold(true).Render(step) +
			" · " + Paint(Accent).Render(flowName),
		paneGutter + "  " + Paint(Live).Render(now),
	}

	if t.CurrentThought != "" {
		out = append(out, prose(t.CurrentThought, w, paneGutter+"  ")...)
	}

	if t.ToolCallCount > 0 {
		out = append(out, paneGutter+"  "+Text(Secondary).Render(
			p.P("overview.tools", t.ToolCallCount, "{n} tool call", "{n} tool calls")))
	}

	out = append(out, paneGutter+"  "+Text(Tertiary).Render(
		p.T("overview.flow_full_tree_hint", "press [2] for full flow tree")))

	return out
}

func (m Model) waitingFlowCard(t view.Task, flowName string) []string {
	p := m.opts.Words
	stateWord, role := m.stateWord(t)
	step := orDef(t.Phase, stateWord)

	return []string{
		paneGutter + Paint(role).Render("⏸ ") + Text(Primary).Bold(true).Render(step) +
			" · " + Paint(Accent).Render(flowName),
		paneGutter + "  " + Paint(role).Render(stateWord),
		paneGutter + "  " + Text(Tertiary).Render(
			p.T("overview.flow_full_tree_hint", "press [2] for full flow tree")),
	}
}

func (m Model) doneFlowCard(t view.Task, flowName string) []string {
	p := m.opts.Words
	mark := Paint(OK).Render("✓ ")
	verdict := p.T("overview.flow_completed", "flow completed successfully")

	if t.Reason.Key == view.ReasonFailed {
		mark = Paint(Bad).Render("✗ ")
		verdict = p.T("overview.flow_failed", "flow stopped on failure")
	}

	out := []string{
		paneGutter + mark + Text(Primary).Bold(true).Render(verdict) +
			" · " + Paint(Accent).Render(flowName),
	}

	if done := m.finishedPhases(); len(done) > 0 {
		out = append(out, paneGutter+"  "+Text(Secondary).Render(
			p.P("overview.flow_finished_phases", len(done),
				"{n} phase executed", "{n} phases executed")))
	}

	out = append(out, paneGutter+"  "+Text(Tertiary).Render(
		p.T("overview.flow_full_tree_hint", "press [2] for full flow tree")))

	return out
}

func (m Model) todoFlowCard(flowName string) []string {
	p := m.opts.Words

	return []string{
		paneGutter + Paint(Dim).Render("○ ") + Text(Primary).Bold(true).Render(
			p.T("overview.flow_ready", "ready to start")) + " · " + Paint(Accent).Render(flowName),
		paneGutter + "  " + Text(Tertiary).Render(
			p.T("overview.not_started", "task has not been started yet (press [n] to start)")),
	}
}
