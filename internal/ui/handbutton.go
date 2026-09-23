package ui

// The buttons on the foot of the flow tree: the delivery verbs asked for by
// hand, and the one gesture each of them offers.
//
// A verb that has come back is asked for again. A verb that is still out is
// let go of, which until now it could not be: the node sat on ⚡ with a
// spinner beside it and the only way past was to close the window.

import (
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/panes"
)

// pressedHand answers the button on one of those nodes, numbered the way
// the tree numbers its folds: the phases first, these after.
func (m Model) pressedHand(at int) (tea.Model, tea.Cmd) {
	st, ok := m.handAt(at)
	if !ok {
		return m, nil
	}

	if st.Ended.IsZero() {
		return m.stopWaiting(st.Verb)
	}

	return m.askAgain(st.Verb)
}

// handAt is the delivery verb one of those numbers stands for.
func (m Model) handAt(at int) (panes.Step, bool) {
	f, err := flow.Resolve(m.opts.Flows, m.subject().Flow)
	if err != nil {
		return panes.Step{}, false
	}

	steps := panes.Hands(m.paneEnv(tabFlow))

	j := at - len(f.Phases)
	if j < 0 || j >= len(steps) {
		return panes.Step{}, false
	}

	return steps[j], true
}

// stopWaiting lets go of a verb that has not come back.
//
// What it ends is the wait, and it says so rather than pretending more. The
// engine carrying the verb is in a goroutine this window handed a prompt to
// and kept no handle on, so nothing here kills it: what comes back later
// still lands in the supervisor's thread, where every other answer lands.
// What stops is the part the reader is stuck behind — the spinner, the
// band, and a node that offers nothing.
//
// Only the verb this window is waiting on. A node still out that this
// cockpit never asked for was asked somewhere else, and stopping a wait
// nobody here is holding would write an ending onto work that is running.
func (m Model) stopWaiting(verb string) (tea.Model, tea.Cmd) {
	p := m.opts.Words
	if !sameVerb(m.delivering.verb, verb) {
		return m.say(p.T("hand.not_ours", "{verb} was asked for somewhere else, so this window cannot stop it",
			about("verb", verb))), nil
	}

	// The cause the record keeps. The verb did not break, the reader gave
	// up on it, and the sentence under the tree's red node is what says
	// which of the two happened.
	gaveUp := errors.New(p.T("hand.gave_up",
		"stopped from the cockpit: nobody waited for {verb} to answer", about("verb", verb)))

	m.supervisorBusy = false
	m = m.answered("", gaveUp)

	return m.say(p.T("hand.stopped", "nothing is waiting on {verb} any more", about("verb", verb))), nil
}

// askAgain hands the same verb to the supervisor a second time, through the
// door its key presses.
//
// The captions are the ones the record kept, paired here with the gesture
// that wrote them. Written out rather than derived, for the reason the
// deliver grid writes its own pairs out: the caption is a string in a log
// and the gesture is a method, and nothing in between maps one to the other.
func (m Model) askAgain(verb string) (tea.Model, tea.Cmd) {
	again := map[string]func() (tea.Model, tea.Cmd){
		"CREATE PR":        m.deliverPR,
		"UPDATE PR":        m.updatePRBranch,
		"FIX CHECKS":       m.fixChecks,
		"MORE TESTS":       m.addMoreTests,
		"RESOLVE COMMENTS": m.resolveComments,
		"DEEP REVIEW":      m.reviewPR,
	}[strings.ToUpper(strings.TrimSpace(verb))]

	if again == nil {
		return m.say(m.opts.Words.T("hand.no_gesture",
			"{verb} is not something this window can ask for again", about("verb", verb))), nil
	}

	return again()
}

// sameVerb compares two captions the way the record and the window spell
// them, which is the same word in two different cases often enough to matter.
func sameVerb(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
