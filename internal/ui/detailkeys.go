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
	// The base is one of the things an open forgets: it belongs to the
	// repository this task is in, and asking for it again is the one thing
	// this window does per open rather than per tick.
	m.diffBase, m.diffAsking = baseRef{}, true
	for i := range m.panes {
		m.panes[i] = viewport.New()
	}

	return m.syncPanes(), tea.Batch(logOf(m.opts.Reader, t), filesOf(m.opts.Reader, t), diffOf(m.opts.Reader, t, m.diffBase))
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
	case key.Matches(k, m.keys.Menu), k.String() == "m":
		return m.openMenuForContext(), nil
	case k.String() == "M":
		return m.mergePR()
	case k.String() == "X":
		return m.closePR()
	case k.String() == "u" || k.String() == "U":
		return m.updatePRBranch()
	case k.String() == "e" || k.String() == "w" || k.String() == "W":
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
	case k.String() == "t":
		m = m.cycleThinking()

		thk := m.knobs.Thinking
		if thk == "" {
			thk = "adaptive"
		}

		p := m.opts.Words

		return m.syncPanes().say(p.T("detail.thinking_changed",
			"thinking mode set to {mode}", about("mode", thk))), nil
	case k.String() == "k" || k.String() == "K":
		return m.openEngines(), nil
	case k.String() == "E":
		m = m.cycleEffort()

		eff := m.knobs.Effort
		if eff == "" {
			eff = "high"
		}

		p := m.opts.Words

		return m.syncPanes().say(p.T("detail.effort_changed",
			"effort level set to {effort}", about("effort", eff))), nil
	case k.String() == "F":
		return m.openFlows(), nil
	// The three keys the needs-you banner names are r, a and c, and all
	// three are answered here. The diff tab's own r is matched above, so a
	// reader reading a change still toggles the rationale with it.
	case key.Matches(k, m.keys.Resume):
		return m.verbOn(m.subject(), m.keys.Resume, "resume")
	case key.Matches(k, m.keys.Skip):
		return m.askSkip()
	case key.Matches(k, m.keys.Ask):
		return m.openNote(), nil
	case key.Matches(k, m.keys.CLI):
		return m.launchInteractiveCLI()
	case key.Matches(k, m.keys.Open), key.Matches(k, m.keys.Last):
		return m.newest(), nil
	case key.Matches(k, m.keys.Help):
		return m.openHelp(), nil
	case key.Matches(k, m.keys.Quit):
		return m, tea.Quit
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

// filesOf reads what one task's directory holds, off the event loop.
func filesOf(r Reader, t view.Task) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return filesMsg{ID: t.ID, Err: errNoRecordPort}
		}

		files, err := r.Files(t.RepoPath, t.ID)
		if err != nil {
			return filesMsg{ID: t.ID, Err: err}
		}

		return filesMsg{ID: t.ID, Files: files}
	}
}

// fileTextOf reads one file of a task's directory, off the event loop.
func fileTextOf(r Reader, t view.Task, name string) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return fileTextMsg{ID: t.ID, Name: name, Err: errNoRecordPort}
		}

		text, err := r.FileText(t.RepoPath, t.ID, name)
		if err != nil {
			return fileTextMsg{ID: t.ID, Name: name, Err: err}
		}

		return fileTextMsg{ID: t.ID, Name: name, Text: text}
	}
}

// logOf reads one task's record, off the event loop.
func logOf(r Reader, t view.Task) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return logMsg{ID: t.ID, Err: errNoRecordPort}
		}

		entries, err := r.Log(t.RepoPath, t.ID)
		if err != nil {
			return logMsg{ID: t.ID, Err: err}
		}

		return logMsg{ID: t.ID, Entries: entries}
	}
}
