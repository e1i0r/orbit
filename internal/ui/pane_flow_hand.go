package ui

// The foot of the flow tree: what the operator asked for by hand.
//
// The phases above it are the flow the task was started under, and this is
// everything a delivery key asked for afterwards — open the pull request,
// make its checks pass, answer its reviews. They belong on the same tree
// because they are the same question: what has been done about this task,
// in the order it was done. They are not phases and are drawn apart from
// them, because nothing in a flow decided they would happen.

import (
	"fmt"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// handStep is one delivery verb as the tree draws it: what was asked for,
// what was handed the work, whether it has come back, and what it said.
type handStep struct {
	verb   string
	by     string
	at     time.Time
	done   bool
	failed bool
	text   string
	cause  string
	took   string
}

// byHand reads the delivery verbs out of the record, oldest first, each with
// the answer that closed it.
//
// An answer is matched to the last ask of the same verb that is still open,
// rather than to the ask before it: two verbs can be out at once — the
// window waits on one at a time, but a run of the same task, or a second
// cockpit, is not asked. Pairing by verb keeps them apart, and an answer
// that pairs with nothing at all is a verb asked for before this build knew
// how to write it down.
func (m Model) byHand() []handStep {
	var steps []handStep

	for _, e := range m.entries {
		if e.What() == view.EntryDeliverAsked {
			steps = append(steps, handStep{verb: e.Verb, by: e.By, at: e.At})
			continue
		}

		if e.What() != view.EntryDeliverAnswered {
			continue
		}

		for i := len(steps) - 1; i >= 0; i-- {
			if steps[i].verb != e.Verb || steps[i].done {
				continue
			}

			steps[i].done = true
			steps[i].failed = e.Cause != ""
			steps[i].text, steps[i].cause = e.Said(), e.Cause

			if !steps[i].at.IsZero() && !e.At.IsZero() {
				steps[i].took = elapsed(e.At, steps[i].at)
			}

			break
		}
	}

	return steps
}

// handNode is one of those steps on the tree, in the shape a phase has: the
// branch it hangs off, where it got to, and — once opened — what was handed
// the work and what came back.
func (m Model) handNode(st handStep, i int, last bool) []string {
	branch, subBranch := "├──", "│  "
	if last {
		branch, subBranch = "└──", "   "
	}

	icon, status, role := m.handStanding(st)
	mark := Text(Tertiary).Render(foldMark(m.rowOpen(tabFlow, i)))

	head := fmt.Sprintf("  %s %s%s %s · %s",
		Paint(Dim).Render(branch), mark, icon, Paint(role).Bold(true).Render(st.verb), status)
	if st.took != "" {
		head += " " + Paint(Dim).Render(fmt.Sprintf("(%s)", st.took))
	}

	out := []string{head}
	if m.rowOpen(tabFlow, i) {
		out = append(out, subRows(m.handSubItems(st), subBranch)...)
	}

	return append(out, "  "+Paint(Dim).Render(subBranch))
}

// handStanding is where a delivery verb got to: the glyph, the word, and the
// role both are painted in.
//
// A verb that has not come back is drawn as work in progress and not as
// something pending, which is the whole point of writing the ask down: the
// supervisor is out doing it, and the reader pressed the key minutes ago.
func (m Model) handStanding(st handStep) (string, string, Role) {
	p := m.opts.Words

	switch {
	case st.failed:
		return Paint(Bad).Render("✗"),
			Paint(Bad).Render(p.T("flow.hand_broke", "came back broken")), Bad
	case st.done:
		return Paint(OK).Render("✓"), Paint(OK).Render(p.T("flow.hand_done", "came back")), OK
	default:
		return Paint(Live).Render("⚡"),
			Paint(Live).Bold(true).Render(p.T("flow.hand_out", "asked for, still out")), Live
	}
}

// handSubItems is everything hanging off one of those nodes: who was handed
// it, why it broke, and what it answered.
func (m Model) handSubItems(st handStep) []subItem {
	p := m.opts.Words

	var items []subItem

	if st.by != "" {
		items = append(items, subItem{text: fmt.Sprintf("⚙️ %s: %s",
			p.T("flow.hand_by", "handed to"), st.by)})
	}

	if st.cause != "" {
		items = append(items, subItem{text: fmt.Sprintf("❌ %s: %s",
			Paint(Bad).Bold(true).Render(p.T("flow.tree_error", "error details")),
			Paint(Bad).Render(st.cause))})
	}

	return append(items, m.phaseOutcome(st.text)...)
}

// handOutRows is the row the deliver block grows while a verb is still out:
// what was asked for, what has it, and how long ago it was asked.
//
// It is under those keys rather than on a tab of its own because that is
// where the reader is standing when they wonder. They pressed one of these
// captions, the band said one sentence and moved on, and the work carries on
// for minutes inside an engine with nothing on screen to show for it — which
// is indistinguishable from a key that did nothing at all.
//
// The last one asked for, and not a list: the window waits on one verb at a
// time, and the tab beside this one draws every one of them in order.
func (m Model) handOutRows() []string {
	p := m.opts.Words

	steps := m.byHand()
	for i := len(steps) - 1; i >= 0; i-- {
		st := steps[i]
		if st.done {
			continue
		}

		said := p.T("overview.deliver_out_bare", "{verb} is out", about("verb", st.verb))
		if st.by != "" {
			said = p.T("overview.deliver_out", "{verb} is out with {by}",
				about("verb", st.verb), about("by", st.by))
		}

		if ago := elapsed(m.now, st.at); ago != "" {
			said += " · " + p.T("overview.deliver_ago", "asked {ago} ago", about("ago", ago))
		}

		return []string{paneGutter + Paint(Live).Render("⚡ "+said)}
	}

	return nil
}
