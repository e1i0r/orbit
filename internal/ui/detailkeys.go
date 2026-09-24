package ui

// The task view's key map: the gestures that only exist one level down, and
// the one rule that keeps a live log honest.
//
// It is a file of its own because detail.go is the frame and this is the
// behaviour, and because both were over the line together. The split is the
// same one keypress.go and screen.go already make on the board.

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// openDetail opens the task view on one task, from the board.
func (m Model) openDetail(t view.Task) (Model, tea.Cmd) {
	m.screen, m.detail, m.tab = screenDetail, t.ID, tabOverview
	m.entries, m.logErr, m.diff, m.following = nil, nil, "", true
	m.diffErr, m.diffKnown, m.diffNoBase = nil, false, false
	m.expandedDetail, m.shutAttempts = false, nil
	m.opened = [tabCount]map[int]bool{}
	// Back to the top of the tree, with everything else this open forgets.
	// A cursor kept across tasks would be standing on the fourth phase of
	// a flow the new task does not walk.
	m.tree = treeAt{}
	// The base is one of the things an open forgets: it belongs to the
	// repository this task is in, and asking for it again is the one thing
	// this window does per open rather than per tick.
	// A different task is a different worktree, so the fingerprint in hand
	// says nothing about it: cleared, or the first read of the new task
	// would answer "the same" about the old one's.
	m.diffBase, m.diffClock = baseRef{}, diffClock{asking: true}
	m = m.forgetImpact().forgetComparison().forgetTree()

	for i := range m.panes {
		m.panes[i] = viewport.New()
	}

	// The history is read once per open, the way the base is: it is a fact
	// about the repository this task is in, it takes about as long as the
	// diff, and the mark on the tab strip is only honest if it was read
	// whether or not the reader went looking.
	next, impact := m.syncPanes().askImpact()

	// And the tree with it, for the same reason and at about the same
	// cost: it is one `git ls-files` of a checkout that is already open.
	next, shape := next.askTree()

	return next, tea.Batch(logOf(m.opts.Reader, t), filesOf(m.opts.Reader, t),
		diffOf(m.opts.Reader, t, m.diffBase, ""), impact, shape)
}

// detailKey is the task view's map.
func (m Model) detailKey(k fmt.Stringer) (tea.Model, tea.Cmd) {
	if m.diffFilePicker {
		return m.handleDiffFilePickerKey(k)
	}

	if targetTab, ok := keyToPane(k.String()); ok {
		return m.showTab(targetTab), nil
	}

	switch {
	case m.tab == tabDiff && key.Matches(k, m.keys.Sideways):
		return m.sideways(k), nil
	case m.tab == tabDiff && (k.String() == "]" || k.String() == ">"):
		return m.jumpNextDiffFile(), nil
	case m.tab == tabDiff && (k.String() == "[" || k.String() == "<"):
		return m.jumpPrevDiffFile(), nil
	case m.tab == tabDiff && k.String() == "n":
		return m.jumpNextDiffHunk(), nil
	case m.tab == tabDiff && k.String() == "N":
		return m.jumpPrevDiffHunk(), nil
	case m.tab == tabDiff && k.String() == "f":
		return m.openDiffFilePicker(), nil
	case m.tab == tabDiff && (k.String() == " " || k.String() == "space" || k.String() == "z"):
		return m.toggleCollapseCurrentFile(), nil
	case m.tab == tabOverview && (k.String() == "z" || k.String() == "Z"):
		return m.foldAll(), nil
	case m.tab == tabDiff && k.String() == "Z":
		return m.toggleCollapseAll(), nil
	case m.tab == tabImpact && (k.String() == "r" || k.String() == "R"):
		return m.compareSidesCmd()
	case m.tab == tabDiff && (k.String() == "r" || k.String() == "R"):
		m.hideDiffRationale = !m.hideDiffRationale
		p := m.opts.Words

		msg := p.T("diff.rationale_shown", "💡 LLM decisions and reasoning: visible")
		if m.hideDiffRationale {
			msg = p.T("diff.rationale_hidden", "💡 LLM decisions and reasoning: hidden")
		}

		return m.syncPanes().say(msg), nil
	case key.Matches(k, m.keys.Back):
		m.screen = screenList
		return m, nil
	case key.Matches(k, m.keys.NextTab):
		return m.showTab((m.tab + 1) % tabCount), nil
	case key.Matches(k, m.keys.PrevTab):
		return m.showTab((m.tab + tabCount - 1) % tabCount), nil
	case key.Matches(k, m.keys.Edit):
		return m.edit()
	case key.Matches(k, m.keys.Board):
		// The board's commands are reachable from a task's screen too: what
		// they are about is not the task, and a reader should not have to
		// leave one screen to reach the other.
		return m.openMenu(""), nil
	case key.Matches(k, m.keys.Menu), k.String() == "m":
		return m.openMenuForContext(), nil
	// J and not M: M is the board menu, matched above on every screen.
	case k.String() == "J":
		return m.mergePR()
	case k.String() == "X":
		return m.closePR()
	// U alone: u is the prompt pane's letter, taken by keyToPane above.
	case k.String() == "U":
		return m.updatePRBranch()
	// w and W are not here: they are the thinking pane's own key, taken by
	// keyToPane above, so this case never saw them.
	case k.String() == "e":
		m.expandedDetail = !m.expandedDetail
		p := m.opts.Words

		msg := p.T("detail.mode_expanded", "expanded view (all fields unwrapped)")
		if !m.expandedDetail {
			msg = p.T("detail.mode_compact", "compact view (single-line summary)")
		}

		return m.syncPanes().say(msg), nil
	case k.String() == "v" || k.String() == "V":
		m.rawText = !m.rawText
		p := m.opts.Words

		msg := p.T("detail.mode_markdown", "formatted view (markdown)")
		if m.rawText {
			msg = p.T("detail.mode_raw", "plain text view (raw)")
		}

		return m.syncPanes().say(msg), nil
	case k.String() == "p" || k.String() == "P":
		return m.deliverPR()
	// Fix checks is C, and lower-case c is left to the interactive CLI.
	// Both were c: the CLI is a binding the key bar draws as [c]
	// interactive CLI on this screen and on the board, this case is
	// matched by the letter and sits above it, and so the hint the reader
	// clicked wrote an instruction note for the next run instead. Upper
	// case is where the deliver toolbar puts its other verbs — M merge, X
	// close, T more tests — and it costs the CLI only its alias, since c
	// is what is drawn for it everywhere.
	case k.String() == "C":
		return m.fixChecks()
	case k.String() == "T":
		return m.addMoreTests()
	// R and not r: r lets a parked run go, and a reader who meant to bring
	// back a review would otherwise pass the gate it is waiting at.
	case k.String() == "R":
		return m.resolveComments()
	case k.String() == "D":
		return m.reviewPR()
	case k.String() == "F":
		return m.openFlows(), nil
	case key.Matches(k, m.keys.Language):
		return m.switchLanguage()
	case key.Matches(k, m.keys.RetryPhase):
		return m.retryPhase()
	case key.Matches(k, m.keys.Ask):
		return m.openNote(), nil
	// Starting a run is offered on the task's own screen and not only on the
	// board. A task that was abandoned is read here, and here is where the
	// reader decides to set it going again.
	case key.Matches(k, m.keys.Start):
		return m.openStart()
	case key.Matches(k, m.keys.CLI):
		return m.launchInteractiveCLI()
	// The tree is the one pane made of nodes, so on it the arrows walk the
	// nodes and ↵ presses what they stand on. Everywhere else they scroll.
	case m.tab == tabFlow && key.Matches(k, m.keys.Up):
		return m.stepTree(-1)
	case m.tab == tabFlow && key.Matches(k, m.keys.Down):
		return m.stepTree(1)
	case m.tab == tabFlow && key.Matches(k, m.keys.Open):
		return m.pressTree()
	case key.Matches(k, m.keys.Open), key.Matches(k, m.keys.Last):
		return m.newest(), nil
	case key.Matches(k, m.keys.Help):
		return m.armTip(), nil
	case key.Matches(k, m.keys.Quit):
		return m, tea.Quit
	}

	// What is left of the verbs about a task, r and x among them, and of
	// the window's own keys: the ones this screen has not taken.
	if next, cmd, ok := m.leftOver(k); ok {
		return next, cmd
	}

	return m.scroll(k), nil
}

// showTab puts one pane on top.
func (m Model) showTab(t tab) Model {
	m.tab = t
	return m
}

// scroll moves the pane, and is the one site the follow rule lives at.
func (m Model) scroll(k fmt.Stringer) Model {
	vp := m.panes[m.tab]
	was := vp.YOffset()

	switch {
	case key.Matches(k, m.keys.Up):
		vp.ScrollUp(1)
	case key.Matches(k, m.keys.Down):
		vp.ScrollDown(1)
	case key.Matches(k, m.keys.PageUp):
		vp.ScrollUp(m.frame.Body.H)
	case key.Matches(k, m.keys.PageDown):
		vp.ScrollDown(m.frame.Body.H)
	case key.Matches(k, m.keys.First):
		vp.GotoTop()
	case key.Matches(k, m.keys.Last):
		vp.GotoBottom()
	default:
		return m
	}

	m.panes[m.tab] = vp
	if m.tab == tabTimeline {
		if vp.AtBottom() {
			m.following = true
		} else if vp.YOffset() < was {
			m.following = false
		}
	}

	return m
}

// sideways scrolls the pane along a line too wide for it, which only the
// diff ever is.
func (m Model) sideways(k fmt.Stringer) Model {
	vp := m.panes[m.tab]
	if k.String() == "left" {
		vp.ScrollLeft(sidewaysStep)
	} else {
		vp.ScrollRight(sidewaysStep)
	}

	m.panes[m.tab] = vp

	return m
}

// sidewaysStep is how far one press moves along a line.
const sidewaysStep = 8

// newest jumps to the end of the pane and arms the tail again.
func (m Model) newest() Model {
	vp := m.panes[m.tab]
	vp.GotoBottom()

	m.panes[m.tab] = vp
	if m.tab == tabTimeline {
		m.following = true
	}

	return m
}
