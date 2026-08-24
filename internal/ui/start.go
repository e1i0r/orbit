package ui

// The start dialog: what a run will be, decided before anything runs.
//
// It is one screen and not a wizard. Everything that decides a run is on it
// at once — the flow, the phases that flow means, and the standing switch
// that decides whether those phases stop for anybody — because the question
// a reader is answering is "is this the right thing to spend an hour on",
// and that question cannot be answered one field at a time.
//
// The flow line is the first line and it is the one that changes the other
// three: f cycles it, the phases underneath are redrawn from whatever that
// name resolved to, and ⏎ starts what is on screen. Choosing a different one
// here is the recorded override — task.started carries the flow a run walked
// — and not a rewrite of the task.
//
// *Deliberately left out*, and this is where a reader will look for it: the
// specified screen has space to hold a phase back and e to edit one, and
// neither is built. The reason is the wire, not the effort. A run is started
// by spawning `orbit run -repo … -flow <name> <id>`, and a name is the whole
// of what that command line carries — a flow with one phase held back, or
// with a different model on phase two, has nothing to travel in. Building
// the gestures anyway would put a choice on screen that the runner never
// hears: the reader holds a phase back, presses ⏎, and watches it run. That
// is worse than the two keys not being there. They arrive when a run can be
// started from a flow rather than from a name. Until then the footer cannot
// advertise them, because it is drawn from the bindings this file matches
// against.

import (
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
)

// namedInRefusal is how many waiting tasks the cap's refusal lists by name.
//
// Three, because the sentence has to fit an activity band one row tall and
// because three ids is enough to recognise which pile is meant. The count is
// in the same sentence, so a reader is never left thinking three is all
// there are.
const namedInRefusal = 3

// startFlow is one entry in the cycle: a name, where it came from, and what
// that name resolved to.
//
// The error is carried rather than dropped, because a flow of the reader's
// own that will not parse is still a flow they have, and this screen is
// where they should find that out — before a worktree, a process and a bill.
// It is the same argument flow.Validate makes about loading one at all.
type startFlow struct {
	name   string
	origin flow.Origin // built in, the reader's own, or shadowing a built-in
	flow   flow.Flow
	err    error
}

// startModel is what the dialog remembers: which task it is about, every
// flow it can offer, and which of them is showing.
//
// The whole cycle is resolved when the dialog opens rather than one flow at
// a time, so that f costs no reading at all. A key that stats a directory is
// a key that stutters on a slow disk, and this one is pressed to compare.
type startModel struct {
	id    string
	flows []startFlow
	at    int
}

// chosen is the flow on screen, and the one ⏎ starts.
func (s startModel) chosen() startFlow {
	if len(s.flows) == 0 {
		return startFlow{}
	}
	return s.flows[s.at%len(s.flows)]
}

// cycle is the flows in the order f visits them, ending on the one showing.
// "quick · task · careful" is read as "press f twice more and you are back".
func (s startModel) cycle() []startFlow {
	if len(s.flows) == 0 {
		return nil
	}
	at := s.at % len(s.flows)
	return append(slices.Clone(s.flows[at+1:]), s.flows[:at+1]...)
}

// newStart builds the dialog's state for one task.
//
// The built-ins and the reader's own are sorted into one list rather than
// shown as two, because at the moment of starting a run the difference that
// matters is what a flow does and not where it came from — which is what the
// mark beside the name is for, in the words `orbit flows` uses. A task
// written against a name that is neither is kept at the front all the same:
// a name that will not resolve is a thing to say on this screen, not a task
// to hide from it, and it arrives with flow.OriginUnknown because that is
// what it is: a name nothing answers to.
//
// Which mark a name gets is flow.List's answer and is not worked out again
// here. It used to be — from flow.UserNames and a slices.Contains against
// the built-ins — and that made two implementations of one rule, one of
// which `orbit flows` used and one of which this dialog used. They agreed
// only because nobody had edited either of them yet.
func newStart(src flow.Source, t view.Task) startModel {
	own := t.Flow
	if own == "" {
		own = flow.Default
	}
	listed := flow.List(src)
	if !slices.ContainsFunc(listed, func(l flow.Listed) bool { return l.Name == own }) {
		listed = append([]flow.Listed{{Name: own}}, listed...)
	}
	s := startModel{id: t.ID, flows: make([]startFlow, 0, len(listed))}
	for i, l := range listed {
		if l.Name == own {
			s.at = i
		}
		f, err := flow.Resolve(src, l.Name)
		s.flows = append(s.flows, startFlow{
			name:   l.Name,
			origin: l.Origin,
			flow:   f,
			err:    err,
		})
	}
	return s
}

// openStart is what n does: put the dialog in front of the row under the
// cursor, or say why there is nothing to put it in front of.
//
// A band header is left alone in silence, and that is the one place in this
// window where a key does nothing without saying so. It is not an oversight:
// ⏎ on a header folds the band, so a reader on a header is somewhere they
// arrived deliberately, and a sentence about tasks would be about rows they
// can see are not there.
func (m Model) openStart() (tea.Model, tea.Cmd) {
	p := m.opts.Words
	r, ok := m.selected()
	if !ok {
		return m.say(p.T("start.nothing_to_start",
			"there is no task here to start; write one with `orbit new`")), nil
	}
	if r.head {
		return m, nil
	}
	if r.task.Live {
		return m.say(p.T("start.already_running", "{id} is already running; press x to stop it first",
			about("id", r.task.ID))), nil
	}
	m.screen, m.start = screenStart, newStart(m.opts.Flows, r.task)
	return m, nil
}

// startKey is the dialog's own map, and it is short on purpose: four keys,
// and every one of them is in the footer.
func (m Model) startKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.screen, m.start = screenList, startModel{}
		return m, nil
	case key.Matches(msg, m.keys.Run):
		return m.runIt()
	case key.Matches(msg, m.keys.ChangeFlow):
		return m.cycleFlow(), nil
	case key.Matches(msg, m.keys.Autopilot):
		return m.autopilot(), nil
	case key.Matches(msg, m.keys.Help):
		// The bar prints [?] on every screen, because help and quit are the
		// two things a reader who is lost reaches for and barLine never
		// drops them. A key the bar offers has to do something, and what ?
		// does everywhere in this window is say it is not built yet.
		return m.notBuilt(m.keys.Help), nil
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}
	return m, nil
}

// cycleFlow moves to the next flow in the cycle, wrapping.
func (m Model) cycleFlow() Model {
	if n := len(m.start.flows); n > 1 {
		m.start.at = (m.start.at + 1) % n
	}
	return m
}

// runIt starts the flow on screen, or refuses and says why.
//
// The cap is checked here as well as inside task.Start, and the duplication
// is deliberate: this one names the tasks that are waiting, which needs the
// board the reader is looking at, and the one inside task.Start is the
// authoritative one — it reads the settings file rather than a board that may
// be half a second old. When they disagree the second answer wins and is
// repeated verbatim, because it is the one that actually decided.
func (m Model) runIt() (tea.Model, tea.Cmd) {
	p := m.opts.Words
	t, ok := m.task(m.start.id)
	if !ok {
		return m.say(p.T("start.gone", "{id} has left the board; nothing was started",
			about("id", m.start.id))), nil
	}
	chosen := m.start.chosen()
	if chosen.err != nil {
		return m.say(chosen.err.Error()), nil
	}
	// The board is walked once. The unread tasks are the count, the
	// colour and the ids in the refusal all at once, and reading it three
	// times to answer one question was three answers that could disagree
	// if a refresh landed between them.
	waiting := board.Unreads(m.board)
	if m.atUnreadCap(len(waiting)) {
		return m.say(m.unreadRefusal(waiting)), nil
	}
	m.screen, m.start = screenList, startModel{}
	return m, start(m.opts.Start, t, chosen.name, len(waiting))
}

// atUnreadCap is whether the brake is on: as many finished tasks unread as
// the settings file allows, or more. A cap of zero is no cap, which is what
// a window opened without a settings file has.
//
// It takes the count rather than reading the board, for two reasons. A
// caller that already holds the unread tasks does not walk the board again
// to be told how many it found. And this is the only spelling of the
// predicate in this package: the header's warning colour and the ToDo band's
// hint had each inlined `limit > 0 && unread >= limit` at the site, so the
// rule that decides whether anything may start was written three times on
// three screens. There is a fourth in internal/task, and that one stays —
// it reads the settings file rather than a board, and it is the one that
// actually decides.
func (m Model) atUnreadCap(unread int) bool {
	limit := m.unreadCap()
	return limit > 0 && unread >= limit
}

// unreadRefusal is the sentence the brake says, and it names names.
//
// A brake that says only "no" is a brake people route around. The ids are
// what turn "the cap is reached" into a next move — press d on one of these
// — and they come from board.Unreads so that the number in the header and
// the ids in this sentence can never be two different rules.
func (m Model) unreadRefusal(waiting []view.Task) string {
	ids := make([]string, 0, namedInRefusal)
	for _, t := range waiting {
		if len(ids) == namedInRefusal {
			ids = append(ids, "…")
			break
		}
		ids = append(ids, t.ID)
	}
	// esc first, and then d. The refusal is only ever said from this
	// dialog, and the dialog does not handle d — it takes every keystroke
	// while it is up. A sentence that names a key the screen it is printed
	// on will not answer is worse than one that names no key at all.
	return m.opts.Words.P("start.unread_cap", len(waiting),
		"{n} finished task is waiting to be read and the cap is {cap}: {ids} — press esc, then d on it",
		"{n} finished tasks are waiting to be read and the cap is {cap}: {ids} — press esc, then d on one",
		about("n", strconv.Itoa(len(waiting))),
		about("cap", strconv.Itoa(m.unreadCap())),
		about("ids", strings.Join(ids, ", ")))
}

// startBindings is the dialog's footer, and its key map, as one list.
//
// The footer is built from these rather than written out, which is the whole
// guarantee: a key printed under the phases is a key startKey matches on,
// and the two cannot drift apart because there is only one of them. f is
// dropped when there is nothing to cycle to.
func (m Model) startBindings() []key.Binding {
	out := []key.Binding{m.keys.Run}
	if len(m.start.flows) > 1 {
		out = append(out, m.keys.ChangeFlow)
	}
	return append(out, m.keys.Autopilot, m.keys.Back)
}

// startHints is that same list as the bar prints it.
func (m Model) startHints() []barHint {
	out := make([]barHint, 0, len(m.startBindings()))
	for _, b := range m.startBindings() {
		out = append(out, hintFor(b))
	}
	return out
}
