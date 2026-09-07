package panes

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
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
	// checked is whether a loop wrote down a turn of its own under this
	// phase's name. A loop runs no engine — its phases do — so it has no
	// phase.started of its own, and without this it read as pending while
	// it was going round and after it had closed.
	checked bool
}

// execOf is what the record says happened to one phase.
func (e Env) execOf(phaseName string) phaseExec {
	var (
		exec  phaseExec
		start view.Entry
	)

	for _, entry := range e.Entries {
		if !strings.EqualFold(entry.Phase, phaseName) {
			continue
		}

		if entry.What() == view.EntryLoopChecked {
			exec.checked = true
		}

		if entry.What() == view.EntryStarted {
			exec.started = true

			start = entry
			if entry.Engine != "" {
				exec.engine = entry.Engine
			}

			if entry.Model != "" {
				exec.model = entry.Model
			}
		}

		if entry.What() == view.EntryFinished {
			exec.finished = true
			exec.cost = entry.Cost

			exec.text = entry.Said()
			if !start.At.IsZero() && !entry.At.IsZero() {
				exec.duration = cells.Elapsed(entry.At, start.At)
			}
		}

		if entry.What() == view.EntryFailed {
			exec.failed = true
			exec.cause = entry.Cause
			exec.exit = entry.Exit
			exec.cost = entry.Cost

			exec.text = entry.Said()
			if !start.At.IsZero() && !entry.At.IsZero() {
				exec.duration = cells.Elapsed(entry.At, start.At)
			}
		}

		if entry.What() == view.EntryCancelled {
			exec.cancelled = true
			exec.text = entry.Said()
		}

		if entry.What() == view.EntryWaiting {
			exec.waiting = true
			exec.cause = entry.Cause
		}
	}

	return exec
}

// Pipeline is the flow this run was started under, drawn as a tree, and
// beside it which phase each node that folds stands for — laid out in one
// pass for the reason the timeline is.
//
// The tree is the pane and folding does not take it down: a closed node is
// still a branch off the trunk with its standing on it, and what it hides is
// how it was configured and what it said.
func Pipeline(e Env) ([]string, map[int]int) {
	p := e.Words

	t := e.Task
	if e.Gone {
		return []string{"  " + theme.Paint(theme.Dim).Render(
			p.T("detail.gone", "this task is no longer on the board"))}, nil
	}

	if e.FlowFailed != "" {
		return []string{"  " + theme.Paint(theme.Bad).Render(e.FlowFailed)}, nil
	}

	f := e.Flow

	title := theme.Paint(theme.Accent).Bold(true).Render(p.T("flow.tree_title", "Pipeline & Execution Tree") + " · ")
	out := []string{
		"",
		"  " + title + theme.Paint(theme.Live).Render(f.Name),
		"",
	}

	heads := map[int]int{}

	// Every node opens onto something: a phase of a resolved flow always
	// names an engine, because flow.Validate refuses a flow whose phases do
	// not, so there is no node whose whole content is its own head.
	for i, phase := range f.Phases {
		heads[len(out)] = i
		out = append(out, e.flowNode(t, phase, i, len(f.Phases), e.pastPhase(f, i))...)
	}

	// The delivery verbs hang off the same trunk, under a heading of their
	// own: they happened to this task, in this order, and nothing in the
	// flow put them there. Their fold keys carry on where the phases' stop,
	// so opening one is the same gesture as opening a phase.
	steps := e.byHand()
	if len(steps) > 0 {
		out = append(out, "  "+theme.Paint(theme.Accent).Bold(true).Render(p.T("flow.by_hand", "Asked for by hand")))
	}

	for j, st := range steps {
		heads[len(out)] = len(f.Phases) + j
		out = append(out, e.handNode(st, len(f.Phases)+j, j == len(steps)-1)...)
	}

	return out, heads
}

// flowNode is one phase of the tree: the branch it hangs off, what happened
// to it, and — once the reader has opened it — how it was set up and what it
// said.
func (e Env) flowNode(t view.Task, phase flow.Phase, i, total int, past bool) []string {
	branch, subBranch := "├──", "│  "
	if i == total-1 {
		branch, subBranch = "└──", "   "
	}

	ex := e.execOf(phase.Name)
	inFlight := t.Band == view.Running && strings.EqualFold(t.Phase, phase.Name)
	st := e.phaseStanding(ex, where{inFlight: inFlight, past: past})

	open := e.row(i)

	// The arrow stands between the branch and the icon, where a tree's
	// disclosure has always stood.
	mark := theme.Text(theme.Tertiary).Render(cells.Fold(open))

	head := fmt.Sprintf("  %s %s%s [%d/%d] %s · %s",
		theme.Paint(theme.Dim).Render(branch), mark, st.glyph, i+1, total,
		theme.Paint(st.role).Bold(true).Render(phase.Name), st.text)

	if ex.cost > 0 {
		head += " " + theme.Paint(theme.Dim).Render(fmt.Sprintf("($%.4f)", ex.cost))
	}

	if ex.duration != "" {
		head += " " + theme.Paint(theme.Dim).Render(fmt.Sprintf("(%s)", ex.duration))
	}

	out := []string{head}
	if open {
		out = append(out, subRows(e.phaseSubItems(phase, ex), subBranch)...)
	}

	// The trunk carries on past the node whether it is open or shut, which
	// is what keeps a folded tree a tree.
	return append(out, "  "+theme.Paint(theme.Dim).Render(subBranch))
}
