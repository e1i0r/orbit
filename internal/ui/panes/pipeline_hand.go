package panes

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

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/prose"
	"github.com/e1i0r/orbit/internal/ui/theme"
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
func (e Env) byHand() []handStep {
	var steps []handStep

	for _, entry := range e.Entries {
		if entry.What() == view.EntryDeliverAsked {
			steps = append(steps, handStep{verb: entry.Verb, by: entry.By, at: entry.At})
			continue
		}

		if entry.What() != view.EntryDeliverAnswered {
			continue
		}

		for i := len(steps) - 1; i >= 0; i-- {
			if steps[i].verb != entry.Verb || steps[i].done {
				continue
			}

			steps[i].done = true
			steps[i].failed = entry.Cause != ""
			steps[i].text, steps[i].cause = entry.Said(), entry.Cause

			if !steps[i].at.IsZero() && !entry.At.IsZero() {
				steps[i].took = cells.Elapsed(entry.At, steps[i].at)
			}

			break
		}
	}

	return steps
}

// handNode is one of those steps on the tree, in the shape a phase has: the
// branch it hangs off, where it got to, and — once opened — what was handed
// the work and what came back.
func (e Env) handNode(st handStep, i int, last bool) []string {
	branch, subBranch := "├──", "│  "
	if last {
		branch, subBranch = "└──", "   "
	}

	standing := e.handStanding(st)
	fold := theme.Text(theme.Tertiary).Render(cells.Fold(e.row(i)))

	head := fmt.Sprintf("  %s %s%s %s · %s",
		theme.Paint(theme.Dim).Render(branch), fold, standing.glyph,
		theme.Paint(standing.role).Bold(true).Render(st.verb), standing.text)
	if st.took != "" {
		head += " " + theme.Paint(theme.Dim).Render(fmt.Sprintf("(%s)", st.took))
	}

	out := []string{head}
	if e.row(i) {
		out = append(out, subRows(e.handSubItems(st), subBranch)...)
	}

	return append(out, "  "+theme.Paint(theme.Dim).Render(subBranch))
}

// handStanding is where a delivery verb got to: the glyph, the word, and the
// role both are painted in.
//
// A verb that has not come back is drawn as work in progress and not as
// something pending, which is the whole point of writing the ask down: the
// supervisor is out doing it, and the reader pressed the key minutes ago.
func (e Env) handStanding(st handStep) standing {
	p := e.Words

	switch {
	case st.failed:
		return standing{
			glyph: theme.Paint(theme.Bad).Render("✗"),
			text:  theme.Paint(theme.Bad).Render(p.T("flow.hand_broke", "came back broken")),
			role:  theme.Bad,
		}
	case st.done:
		return standing{
			glyph: theme.Paint(theme.OK).Render("✓"),
			text:  theme.Paint(theme.OK).Render(p.T("flow.hand_done", "came back")),
			role:  theme.OK,
		}
	default:
		return standing{
			glyph: theme.Paint(theme.Live).Render("⚡"),
			text:  theme.Paint(theme.Live).Bold(true).Render(p.T("flow.hand_out", "asked for, still out")),
			role:  theme.Live,
		}
	}
}

// handSubItems is everything hanging off one of those nodes: who was handed
// it, why it broke, and what it answered.
func (e Env) handSubItems(st handStep) []subItem {
	p := e.Words

	var items []subItem

	if st.by != "" {
		items = append(items, subItem{text: fmt.Sprintf("⚙️ %s: %s",
			p.T("flow.hand_by", "handed to"), st.by)})
	}

	if st.cause != "" {
		items = append(items, subItem{text: fmt.Sprintf("❌ %s: %s",
			theme.Paint(theme.Bad).Bold(true).Render(p.T("flow.tree_error", "error details")),
			theme.Paint(theme.Bad).Render(st.cause))})
	}

	return append(items, e.phaseOutcome(st.text)...)
}

// A Step is a delivery verb that was asked for by hand: what was asked for,
// what was handed the work, and when.
type Step struct {
	Verb string
	By   string
	At   time.Time
}

// Waiting is the last verb asked for that has not come back, and whether
// there is one.
//
// The last one and not a list: the window waits on one verb at a time, and
// the flow tree draws every one of them in order. The band asks this too —
// a verb still out is the one thing about a task that is happening
// somewhere the reader cannot see.
func Waiting(e Env) (Step, bool) {
	steps := e.byHand()
	for i := len(steps) - 1; i >= 0; i-- {
		if st := steps[i]; !st.done {
			return Step{Verb: st.verb, By: st.by, At: st.at}, true
		}
	}

	return Step{}, false
}

// HandOut is the row the deliver block grows while a verb is still out:
// what was asked for, what has it, and how long ago it was asked.
//
// It is under those keys rather than on a tab of its own because that is
// where the reader is standing when they wonder. They pressed one of these
// captions, the band said one sentence and moved on, and the work carries on
// for minutes inside an engine with nothing on screen to show for it — which
// is indistinguishable from a key that did nothing at all.
func HandOut(e Env) []string {
	st, out := Waiting(e)
	if !out {
		return nil
	}

	p := e.Words

	said := p.T("overview.deliver_out_bare", "{verb} is out", about("verb", st.Verb))
	if st.By != "" {
		said = p.T("overview.deliver_out", "{verb} is out with {by}",
			about("verb", st.Verb), about("by", st.By))
	}

	if ago := cells.Elapsed(e.Now, st.At); ago != "" {
		said += " · " + p.T("overview.deliver_ago", "asked {ago} ago", about("ago", ago))
	}

	return []string{prose.Gutter + theme.Paint(theme.Live).Render("⚡ "+said)}
}
