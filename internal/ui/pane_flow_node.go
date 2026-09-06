package ui

// What a phase's node says about where it got to, and the rows that hang
// off it once a reader opens one.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
)

// phaseStanding is where a phase got to: the glyph it is marked with, the
// word for it, and the role both are painted in.
func (m Model) phaseStanding(ex phaseExec, inFlight, past bool) (string, string, Role) {
	p := m.opts.Words

	switch {
	case ex.failed:
		return Paint(Bad).Render("✗"), Paint(Bad).Render(p.T("flow.step_status_failed", "failed")), Bad
	case ex.cancelled:
		return Paint(Warn).Render("⏹"), Paint(Warn).Render(p.T("flow.step_status_cancelled", "cancelled")), Warn
	case ex.waiting:
		return Paint(Warn).Render("⚠️"), Paint(Warn).Render(p.T("flow.step_status_waiting", "waiting at gate")), Warn
	case ex.finished:
		return Paint(OK).Render("✓"), Paint(OK).Render(p.T("flow.step_status_done", "completed")), OK
	case past:
		// A phase the run has gone past is done, whatever it wrote down
		// about itself. A loop writes no phase.finished of its own — what
		// ran are the phases inside it — so the only thing that says its
		// block is closed is that the flow moved on.
		return Paint(OK).Render("✓"), Paint(OK).Render(p.T("flow.step_status_done", "completed")), OK
	case ex.checked:
		return Paint(Live).Render("⚡"),
			Paint(Live).Bold(true).Render(p.T("flow.step_status_looping", "going round")), Live
	case inFlight:
		return Paint(Live).Render("⚡"),
			Paint(Live).Bold(true).Render(p.T("flow.step_status_in_flight", "in progress")), Live
	default:
		return Paint(Dim).Render("○"), Paint(Dim).Render(p.T("flow.step_status_pending", "pending")), Dim
	}
}

// subItem is one row hanging off a node: what it says, and whether it starts
// something or carries on the row before it.
type subItem struct {
	text string
	cont bool
}

// subRows draws what hangs off one node of the tree, under the trunk the node
// left behind it. Both kinds of node — a phase of the flow and a verb the
// operator asked for — are opened onto the same way, and a second copy of
// this loop is a second opinion about where a branch closes.
func subRows(items []subItem, subBranch string) []string {
	last := lastStart(items)
	out := make([]string, 0, len(items))

	for j, item := range items {
		// A row that carries on the row above it hangs off nothing of its
		// own: a branch in front of it says a second thing is under this
		// node, and there is only the one.
		sub := "├──"

		switch {
		case item.cont:
			sub = "   "
		case j == last:
			sub = "└──"
		}

		out = append(out, fmt.Sprintf("  %s %s %s",
			Paint(Dim).Render(subBranch), Paint(Dim).Render(sub), item.text))
	}

	return out
}

// lastStart is the index of the final row that starts something, which is the
// row the branch closes on. The rows after it, if any, are its own tail.
func lastStart(items []subItem) int {
	for i := len(items) - 1; i >= 0; i-- {
		if !items[i].cont {
			return i
		}
	}

	return -1
}

// phaseSubItems is everything hanging off one node: how it was set up, what
// it has to pass, why it broke, and what it wrote.
func (m Model) phaseSubItems(phase flow.Phase, ex phaseExec) []subItem {
	p := m.opts.Words

	var items []subItem

	if cfg := phaseConfig(phase, ex); len(cfg) > 0 {
		items = append(items, subItem{text: fmt.Sprintf("⚙️ %s: %s",
			p.T("flow.tree_engine", "engine"), strings.Join(cfg, " · "))})
	}

	for _, g := range phase.Gates {
		items = append(items, subItem{text: fmt.Sprintf("🚪 %s [%s]: %s",
			p.T("flow.tree_gate", "gate"), g.Name, g.Command)})
	}

	if ex.failed {
		errMsg := ex.cause
		if errMsg == "" && ex.exit != "" {
			errMsg = p.T("flow.exit_code", "exit code: {code}", about("code", ex.exit))
		}

		if errMsg != "" {
			items = append(items, subItem{text: fmt.Sprintf("❌ %s: %s",
				Paint(Bad).Bold(true).Render(p.T("flow.tree_error", "error details")),
				Paint(Bad).Render(errMsg))})
		}
	}

	return append(items, m.phaseOutcome(ex.text)...)
}

// phaseConfig is the dials the phase ran on: what the record says it was,
// and what the flow asked for where the record is silent.
func phaseConfig(phase flow.Phase, ex phaseExec) []string {
	engName := phase.Engine
	if ex.engine != "" {
		engName = ex.engine
	}

	modelName := phase.Model
	if ex.model != "" {
		modelName = ex.model
	}

	var cfg []string

	for _, part := range []string{engName, modelName, "effort:" + phase.Effort, "thinking:" + phase.Thinking} {
		if part == "" || strings.HasSuffix(part, ":") {
			continue
		}

		cfg = append(cfg, part)
	}

	return cfg
}

// phaseOutcome is what the phase wrote, wrapped to the pane and labelled on
// its first row. The rows are cut as well as wrapped, because an engine that
// printed a path with nothing to break at would otherwise be set over the
// margin the scroll bar is drawn in.
func (m Model) phaseOutcome(text string) []subItem {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	// The measure the deepest row of the tree leaves: the branches in front
	// of it, and the label the first row carries.
	measure := max(20, m.frame.Body.W-24)

	var out []subItem

	for _, l := range strings.Split(strings.TrimSpace(text), "\n") {
		if l = strings.TrimSpace(l); l == "" {
			continue
		}

		for _, wl := range splitIntoLines(l, measure) {
			if out == nil {
				out = append(out, subItem{text: fmt.Sprintf("📋 %s: %s",
					m.opts.Words.T("flow.tree_outcome", "outcome"), fit(wl, measure))})

				continue
			}

			out = append(out, subItem{text: fit(wl, measure), cont: true})
		}
	}

	return out
}

// pastPhase is whether the run has gone beyond this phase: a later one has
// started, waited or finished.
//
// The flow is walked in order, so a phase behind the one the run is in is a
// phase that is over — which is the only thing that says a loop's block
// closed, since a loop runs no engine of its own and writes no finish.
func (m Model) pastPhase(f flow.Flow, i int) bool {
	for j := i + 1; j < len(f.Phases); j++ {
		later := m.findPhaseExec(f.Phases[j].Name)
		if later.started || later.finished || later.waiting || later.failed || later.cancelled || later.checked {
			return true
		}
	}

	return false
}
