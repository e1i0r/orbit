package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/prose"
	"github.com/e1i0r/orbit/internal/ui/theme"
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
	flowName := cells.OrDef(t.Flow, flow.Default)
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
	glyph := m.runGlyph(keymap.Working(t))
	step := cells.OrDef(t.Phase, "running")
	now := cells.OrDef(t.CurrentAction, p.T("overview.running_model", "running model..."))

	now = cells.Fit(now, max(20, w-lipgloss.Width(prose.Gutter)-lipgloss.Width(glyph)-4))

	out := []string{
		prose.Gutter + theme.Paint(theme.Live).Render(glyph) + theme.Text(theme.Primary).Bold(true).Render(step) +
			" · " + theme.Paint(theme.Accent).Render(flowName),
		prose.Gutter + "  " + theme.Paint(theme.Live).Render(now),
	}

	if t.CurrentThought != "" {
		out = append(out, prose.Quote(t.CurrentThought, w, prose.Gutter+"  ")...)
	}

	if t.ToolCallCount > 0 {
		out = append(out, prose.Gutter+"  "+theme.Text(theme.Secondary).Render(
			p.P("overview.tools", t.ToolCallCount, "{n} tool call", "{n} tool calls")))
	}

	out = append(out, prose.Gutter+"  "+theme.Text(theme.Tertiary).Render(
		p.T("overview.flow_full_tree_hint", "press [2] for full flow tree")))

	return out
}

func (m Model) waitingFlowCard(t view.Task, flowName string) []string {
	p := m.opts.Words
	stateWord, role := m.stateWord(t)
	step := cells.OrDef(t.Phase, stateWord)

	return []string{
		prose.Gutter + theme.Paint(role).Render("⏸ ") + theme.Text(theme.Primary).Bold(true).Render(step) +
			" · " + theme.Paint(theme.Accent).Render(flowName),
		prose.Gutter + "  " + theme.Paint(role).Render(stateWord),
		prose.Gutter + "  " + theme.Text(theme.Tertiary).Render(
			p.T("overview.flow_full_tree_hint", "press [2] for full flow tree")),
	}
}

func (m Model) doneFlowCard(t view.Task, flowName string) []string {
	p := m.opts.Words
	mark := theme.Paint(theme.OK).Render("✓ ")
	verdict := p.T("overview.flow_completed", "flow completed successfully")

	if t.Reason.Key == view.ReasonFailed {
		mark = theme.Paint(theme.Bad).Render("✗ ")
		verdict = p.T("overview.flow_failed", "flow stopped on failure")
	}

	out := []string{
		prose.Gutter + mark + theme.Text(theme.Primary).Bold(true).Render(verdict) +
			" · " + theme.Paint(theme.Accent).Render(flowName),
	}

	if done := m.finishedPhases(); len(done) > 0 {
		out = append(out, prose.Gutter+"  "+theme.Text(theme.Secondary).Render(
			p.P("overview.flow_finished_phases", len(done),
				"{n} phase executed", "{n} phases executed")))
	}

	out = append(out, prose.Gutter+"  "+theme.Text(theme.Tertiary).Render(
		p.T("overview.flow_full_tree_hint", "press [2] for full flow tree")))

	return out
}

func (m Model) todoFlowCard(flowName string) []string {
	p := m.opts.Words

	return []string{
		prose.Gutter + theme.Paint(theme.Dim).Render("○ ") + theme.Text(theme.Primary).Bold(true).Render(
			p.T("overview.flow_ready", "ready to start")) + " · " + theme.Paint(theme.Accent).Render(flowName),
		prose.Gutter + "  " + theme.Text(theme.Tertiary).Render(
			p.T("overview.not_started", "task has not been started yet (press [n] to start)")),
	}
}
