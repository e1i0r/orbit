package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
)

type phaseExec struct {
	started   bool
	finished  bool
	failed    bool
	cancelled bool
	waiting   bool
	cost      float64
	cause     string
	exit      string
	text      string
	engine    string
	model     string
	duration  string
}

// findPhaseExec finds recorded execution metrics for a phase in m.entries
func (m Model) findPhaseExec(phaseName string) phaseExec {
	var exec phaseExec
	var startEntry view.Entry
	for _, e := range m.entries {
		if !strings.EqualFold(e.Phase, phaseName) {
			continue
		}
		if e.What() == view.EntryStarted {
			exec.started = true
			startEntry = e
			if e.Engine != "" {
				exec.engine = e.Engine
			}
			if e.Model != "" {
				exec.model = e.Model
			}
		}
		if e.What() == view.EntryFinished {
			exec.finished = true
			exec.cost = e.Cost
			exec.text = e.Text
			if !startEntry.At.IsZero() && !e.At.IsZero() {
				exec.duration = elapsed(e.At, startEntry.At)
			}
		}
		if e.What() == view.EntryFailed {
			exec.failed = true
			exec.cause = e.Cause
			exec.exit = e.Exit
			exec.cost = e.Cost
			exec.text = e.Text
			if !startEntry.At.IsZero() && !e.At.IsZero() {
				exec.duration = elapsed(e.At, startEntry.At)
			}
		}
		if e.What() == view.EntryCancelled {
			exec.cancelled = true
			exec.text = e.Text
		}
		if e.What() == view.EntryWaiting {
			exec.waiting = true
			exec.cause = e.Cause
		}
	}
	return exec
}

// flowLines renders Pane 2: Tree view of Flow & Step Results.
func (m Model) flowLines() []string {
	p := m.opts.Words
	t, ok := m.task(m.detail)
	if !ok {
		return []string{"  " + Paint(Dim).Render(p.T("detail.gone", "this task is no longer on the board"))}
	}

	flowName := t.Flow
	if flowName == "" {
		flowName = "quick"
	}
	f, err := flow.Resolve(m.opts.Flows, flowName)
	if err != nil {
		return []string{"  " + Paint(Bad).Render(fmt.Sprintf("flow %q: %v", flowName, err))}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("flow.tree_title", "Pipeline & Execution Tree")+" · ") + Paint(Live).Render(f.Name),
		"",
	}

	totalPhases := len(f.Phases)
	for i, phase := range f.Phases {
		isLast := i == totalPhases-1
		branch := "├──"
		subBranch := "│  "
		if isLast {
			branch = "└──"
			subBranch = "   "
		}

		ex := m.findPhaseExec(phase.Name)
		isInFlight := t.Band == view.Running && strings.EqualFold(t.Phase, phase.Name)

		// Determine step icon and badge
		var icon, statusStr string
		var role Role = Dim
		switch {
		case ex.failed:
			icon = Paint(Bad).Render("✗")
			statusStr = Paint(Bad).Render(p.T("flow.step_status_failed", "failed"))
			role = Bad
		case ex.cancelled:
			icon = Paint(Warn).Render("⏹")
			statusStr = Paint(Warn).Render(p.T("flow.step_status_cancelled", "cancelled"))
			role = Warn
		case ex.waiting:
			icon = Paint(Warn).Render("⚠️")
			statusStr = Paint(Warn).Render(p.T("flow.step_status_waiting", "waiting at gate"))
			role = Warn
		case ex.finished:
			icon = Paint(OK).Render("✓")
			statusStr = Paint(OK).Render(p.T("flow.step_status_done", "completed"))
			role = OK
		case isInFlight:
			icon = Paint(Live).Render("⚡")
			statusStr = Paint(Live).Bold(true).Render(p.T("flow.step_status_in_flight", "in progress"))
			role = Live
		default:
			icon = Paint(Dim).Render("○")
			statusStr = Paint(Dim).Render(p.T("flow.step_status_pending", "pending"))
		}

		// Node header
		nodeHeader := fmt.Sprintf("  %s %s [%d/%d] %s · %s",
			Paint(Dim).Render(branch),
			icon,
			i+1,
			totalPhases,
			Paint(role).Bold(true).Render(phase.Name),
			statusStr,
		)
		if ex.cost > 0 {
			nodeHeader += " " + Paint(Dim).Render(fmt.Sprintf("($%.4f)", ex.cost))
		}
		if ex.duration != "" {
			nodeHeader += " " + Paint(Dim).Render(fmt.Sprintf("(%s)", ex.duration))
		}
		out = append(out, nodeHeader)

		// Sub-tree items
		var subItems []string

		// 1. Engine & config
		engName := phase.Engine
		if ex.engine != "" {
			engName = ex.engine
		}
		modelName := phase.Model
		if ex.model != "" {
			modelName = ex.model
		}
		var cfg []string
		if engName != "" {
			cfg = append(cfg, engName)
		}
		if modelName != "" {
			cfg = append(cfg, modelName)
		}
		if phase.Effort != "" {
			cfg = append(cfg, "effort:"+phase.Effort)
		}
		if phase.Thinking != "" {
			cfg = append(cfg, "thinking:"+phase.Thinking)
		}
		if len(cfg) > 0 {
			subItems = append(subItems, fmt.Sprintf("⚙️ %s: %s",
				p.T("flow.tree_engine", "engine"),
				strings.Join(cfg, " · "),
			))
		}

		// 2. Gates / Verifications
		for _, g := range phase.Gates {
			subItems = append(subItems, fmt.Sprintf("🚪 %s [%s]: %s",
				p.T("flow.tree_gate", "gate"),
				g.Name,
				g.Command,
			))
		}

		// 3. Error details (if failed)
		if ex.failed {
			errMsg := ex.cause
			if errMsg == "" && ex.exit != "" {
				errMsg = "exit code: " + ex.exit
			}
			if errMsg != "" {
				subItems = append(subItems, fmt.Sprintf("❌ %s: %s",
					Paint(Bad).Bold(true).Render(p.T("flow.tree_error", "error details")),
					Paint(Bad).Render(errMsg),
				))
			}
		}

		// 4. Outcome text
		if ex.text != "" {
			lines := strings.Split(strings.TrimSpace(ex.text), "\n")
			for count, l := range lines {
				l = strings.TrimSpace(l)
				if l == "" {
					continue
				}
				if count == 0 {
					subItems = append(subItems, fmt.Sprintf("📋 %s: %s",
						p.T("flow.tree_outcome", "outcome"),
						l,
					))
				} else if count < 3 {
					subItems = append(subItems, fmt.Sprintf("   %s", l))
				}
			}
		}

		// Render sub items with tree branches
		for j, item := range subItems {
			subSym := "├──"
			if j == len(subItems)-1 {
				subSym = "└──"
			}
			out = append(out, fmt.Sprintf("  %s %s %s",
				Paint(Dim).Render(subBranch),
				Paint(Dim).Render(subSym),
				item,
			))
		}

		out = append(out, fmt.Sprintf("  %s", Paint(Dim).Render(subBranch)))
	}

	return out
}
