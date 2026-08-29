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

// findPhaseExec finds recorded execution metrics for a phase in m.entries.
func (m Model) findPhaseExec(phaseName string) phaseExec {
	var (
		exec       phaseExec
		startEntry view.Entry
	)

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

			exec.text = e.Said()
			if !startEntry.At.IsZero() && !e.At.IsZero() {
				exec.duration = elapsed(e.At, startEntry.At)
			}
		}

		if e.What() == view.EntryFailed {
			exec.failed = true
			exec.cause = e.Cause
			exec.exit = e.Exit
			exec.cost = e.Cost

			exec.text = e.Said()
			if !startEntry.At.IsZero() && !e.At.IsZero() {
				exec.duration = elapsed(e.At, startEntry.At)
			}
		}

		if e.What() == view.EntryCancelled {
			exec.cancelled = true
			exec.text = e.Said()
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
	lines, _ := m.flowRows()

	return lines
}

// flowRows is that tree and, beside it, which phase each node that folds
// stands for, laid out in one pass for the reason logRows is.
//
// The tree is the pane and folding does not take it down: a closed node is
// still a branch off the trunk with its standing on it, and what it hides is
// how it was configured and what it said.
func (m Model) flowRows() ([]string, map[int]int) {
	p := m.opts.Words

	t, ok := m.task(m.detail)
	if !ok {
		return []string{"  " + Paint(Dim).Render(
			p.T("detail.gone", "this task is no longer on the board"))}, nil
	}

	flowName := t.Flow
	if flowName == "" {
		flowName = "quick"
	}

	f, err := flow.Resolve(m.opts.Flows, flowName)
	if err != nil {
		return []string{"  " + Paint(Bad).Render(fmt.Sprintf("flow %q: %v", flowName, err))}, nil
	}

	title := Paint(Accent).Bold(true).Render(p.T("flow.tree_title", "Pipeline & Execution Tree") + " · ")
	out := []string{
		"",
		"  " + title + Paint(Live).Render(f.Name),
		"",
	}

	heads := map[int]int{}

	// Every node opens onto something: a phase of a resolved flow always
	// names an engine, because flow.Validate refuses a flow whose phases do
	// not, so there is no node whose whole content is its own head.
	for i, phase := range f.Phases {
		heads[len(out)] = i
		out = append(out, m.flowNode(t, phase, i, len(f.Phases))...)
	}

	return out, heads
}

// flowNode is one phase of the tree: the branch it hangs off, what happened
// to it, and — once the reader has opened it — how it was set up and what it
// said.
func (m Model) flowNode(t view.Task, phase flow.Phase, i, total int) []string {
	branch, subBranch := "├──", "│  "
	if i == total-1 {
		branch, subBranch = "└──", "   "
	}

	ex := m.findPhaseExec(phase.Name)
	inFlight := t.Band == view.Running && strings.EqualFold(t.Phase, phase.Name)
	icon, status, role := m.phaseStanding(ex, inFlight)

	open := m.rowOpen(tabFlow, i)

	// The arrow stands between the branch and the icon, where a tree's
	// disclosure has always stood.
	mark := Text(Tertiary).Render(foldMark(open))

	head := fmt.Sprintf("  %s %s%s [%d/%d] %s · %s",
		Paint(Dim).Render(branch), mark, icon, i+1, total,
		Paint(role).Bold(true).Render(phase.Name), status)

	if ex.cost > 0 {
		head += " " + Paint(Dim).Render(fmt.Sprintf("($%.4f)", ex.cost))
	}

	if ex.duration != "" {
		head += " " + Paint(Dim).Render(fmt.Sprintf("(%s)", ex.duration))
	}

	out := []string{head}

	if items := m.phaseSubItems(phase, ex); open {
		last := lastStart(items)

		for j, item := range items {
			// A row that carries on the row above it hangs off nothing of
			// its own: a branch in front of it says a second thing is under
			// this phase, and there is only the one.
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
	}

	// The trunk carries on past the node whether it is open or shut, which
	// is what keeps a folded tree a tree.
	return append(out, "  "+Paint(Dim).Render(subBranch))
}

// phaseStanding is where a phase got to: the glyph it is marked with, the
// word for it, and the role both are painted in.
func (m Model) phaseStanding(ex phaseExec, inFlight bool) (string, string, Role) {
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
