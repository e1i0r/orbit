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
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/prose"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// handStep is one delivery verb as the tree draws it: what was asked for,
// what was handed the work, whether it has come back, and what it said.
type handStep struct {
	verb string
	by   string
	at   time.Time
	// ended is when the answer came in, which is a different moment from
	// the ask and the one a reader who has just come back is asking about.
	ended  time.Time
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
			steps[i].ended = entry.At
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
			text: theme.Paint(theme.Live).Bold(true).Render(
				p.T("flow.hand_out", "handed over, still working")),
			role: theme.Live,
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
// what was handed the work, when, and — once it is back — what it answered.
type Step struct {
	Verb string
	By   string
	At   time.Time
	// Ended, Said and Cause are empty while the verb is still out. Cause
	// is what broke, and is what tells the two endings apart.
	Ended time.Time
	Said  string
	Cause string
}

// justLanded is how long a verb that has come back stays on the band.
//
// A verb answered an hour ago is history and belongs on the tree; a verb
// answered a minute ago is the answer to "did it finish?", which is the
// question a reader has while they are still looking at the screen.
const justLanded = 3 * time.Minute

// Landed is the verb that has just come back, and whether there is one.
//
// The band said who was working and then went quiet, and quiet is the same
// picture as a key that did nothing: a reader could not tell a pull request
// that had been opened from one that never was. This is the other half of
// StillWorking — it says the work ended, and what it ended as.
func Landed(e Env) (Step, bool) {
	steps := e.byHand()
	for i := len(steps) - 1; i >= 0; i-- {
		st := steps[i]
		if !st.done || st.ended.IsZero() || e.Now.Sub(st.ended) > justLanded {
			continue
		}

		return Step{
			Verb: st.verb, By: st.by, At: st.at,
			Ended: st.ended, Said: st.text, Cause: st.cause,
		}, true
	}

	return Step{}, false
}

// CameBack is what a verb that has come back did: whether it worked, what it
// said it did, and how long ago.
//
// The engine's own sentence rather than one written here. A verb that opened
// a pull request answered with the number; a reader who wanted to know
// whether it finished is told what finished.
func CameBack(p *words.Printer, st Step, now time.Time) string {
	said := p.T("overview.deliver_landed_bare", "{verb} came back", about("verb", st.Verb))

	switch {
	case st.Cause != "":
		said = p.T("overview.deliver_broke", "{verb} came back broken · {cause}",
			about("verb", st.Verb), about("cause", firstLine(st.Cause)))
	case st.Said != "":
		said = p.T("overview.deliver_landed", "{verb} came back · {said}",
			about("verb", st.Verb), about("said", firstLine(st.Said)))
	}

	if ago := cells.Elapsed(now, st.Ended); ago != "" {
		said += cells.Dot + p.T("overview.deliver_since", "{ago} ago", about("ago", ago))
	}

	return said
}

// firstLine is the one line of an answer a band has room for.
func firstLine(said string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(said), "\n")

	return strings.TrimSpace(first)
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

	said := StillWorking(e.Words, st.Verb, st.By, st.At, e.Now)

	return []string{prose.Gutter + theme.Paint(theme.Live).Render("⚡ "+said)}
}

// wondering is how long a verb has to be out before the reader is told, in
// so many words, that nobody is waiting on them.
//
// Under it the sentence stays short, because a verb that has just been asked
// for is obviously working. Past it the reader has had time to wonder, and
// wondering is what cost the ten minutes.
const wondering = 2 * time.Minute

// StillWorking is what a verb that has been handed over and has not come
// back is doing: who has it, how long they have had it, and — once it has
// been long enough to wonder — that nothing is waiting on the reader.
//
// "CREATE PR is out with supervisor" was read as "blocked on a human", and a
// reader spent ten minutes waiting to be asked something that was never
// coming. Nothing was gated: the band said running, the gates were empty and
// orbit_permit had no question to answer. The phase was working. So the
// sentence says working, and says who is doing it.
func StillWorking(p *words.Printer, verb, by string, at, now time.Time) string {
	said := p.T("overview.deliver_out_bare", "{verb} is still working", about("verb", verb))
	if by != "" {
		said = p.T("overview.deliver_out", "{by} is working on {verb}",
			about("verb", verb), about("by", by))
	}

	if ago := cells.Elapsed(now, at); ago != "" {
		said += cells.Dot + p.T("overview.deliver_for", "{ago} so far", about("ago", ago))
	}

	if !at.IsZero() && now.Sub(at) >= wondering {
		said += cells.Dot + p.T("overview.deliver_nothing_waiting", "nothing is waiting on you")
	}

	return said
}
