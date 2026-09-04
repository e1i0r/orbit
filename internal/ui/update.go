package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Update is the whole of the window's behaviour, and every case in it is a
// row of the transition table in update_test.go.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m = m.writeDown(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg.Width, msg.Height), nil
	case tea.BackgroundColorMsg:
		m.dark = msg.IsDark()
		return m, nil
	case tickMsg:
		// The task view's log is on the same clock as the board, and for
		// the same reason: an append-only file that did not grow costs one
		// stat to find that out. A tail that only moved when the reader
		// pressed a key would not be a tail.
		cmds := []tea.Cmd{refresh(m.opts.Reader), tick()}
		if m.screen == screenDetail {
			cmds = append(cmds, logOf(m.opts.Reader, m.subject()), filesOf(m.opts.Reader, m.subject()))
		}

		return m, tea.Batch(cmds...)
	case rescanMsg:
		// The diff rides this clock rather than the log's tickMsg one,
		// because git diff is heavier than the stat a tick costs: at
		// board.RescanEvery this is a quarter the log's cadence, which is
		// slow enough to be cheap and still lets a live task's diff change
		// while the reader is looking at it, rather than only at the
		// moment the view was opened.
		//
		// The clock is slower than one diff's worst case all the same — up
		// to five seconds for the diff itself — so a tick that finds the
		// last one still out at git does not ask again. Without that, a
		// repository slow enough to need the bound is the one that gets a
		// second, third and sixth request piled on top of the first.
		cmds := []tea.Cmd{rescan(m.opts.Reader), rescanTick()}
		if m.screen == screenDetail && !m.diffAsking {
			m.diffAsking = true
			cmds = append(cmds, diffOf(m.opts.Reader, m.subject(), m.diffBase))
		}

		return m, tea.Batch(cmds...)
	case elapsedMsg:
		m.now = time.Time(msg)
		return m, elapsedTick()
	case spinnerTickMsg:
		// The frame that was asked for has landed, so the chain is free to
		// be extended — by this, if anything is still moving, and by
		// nobody else while it is.
		m.now = time.Time(msg)
		m.spinning = false
		m, next := m.nextFrame()

		return m, next
	case upgradeTickMsg:
		return m, tea.Batch(checkUpgradeCmd(m.opts.Version), upgradeTick())
	case boardMsg:
		return m.applyBoard(msg)
	case controlMsg:
		return m.say(m.controlSaid(msg)), nil
	case startedMsg:
		return m.say(m.startedSaid(msg)), nil
	case readMsg:
		return m.say(m.readSaid(msg)), nil
	case requeuedMsg:
		return m.say(m.requeuedSaid(msg)), nil
	case outputMsg:
		// A tick for a run this window already stopped watching, or one
		// whose name does not match what is on screen, is dropped rather
		// than painted; the pump is only re-armed while its own watch is
		// still the one running.
		if m.watching == nil || m.watching.name != msg.Name {
			return m, nil
		}

		m.output = msg.Text

		return m, outputPump(m.watching)
	case commandMsg:
		next := m
		if m.watching != nil && m.watching.name == msg.Name {
			next.output = msg.Text
			next.watching = nil
		}

		if msg.Err != nil {
			return next.say(m.errSaid(msg.Err)), nil
		}

		return next.say(next.commandSaid(msg)), nil
	case sessionMsg:
		return m.session(msg)
	case sessionEndedMsg:
		return m.sessionEnded(msg), nil
	case supervisorReplyMsg:
		m.supervisorBusy = false
		m = m.syncSupervisor()

		m.supervisor.offset = 999999
		if msg.Err != nil {
			return m.say(m.opts.Words.T("supervisor.error", "supervisor error: {err}", about("err", m.errSaid(msg.Err)))), nil
		}

		return m.say(m.opts.Words.T("supervisor.replied", "supervisor replied in thread")), nil
	case cliEndedMsg:
		return m.handleCLIEnded(msg)
	case sessionFiledMsg:
		return m.handleSessionFiled(msg)
	case editorMsg:
		if msg.Err != nil {
			return m.say(m.errSaid(msg.Err)), nil
		}

		return m, nil
	case upgradeAvailableMsg:
		m.upgradeAvailable = msg.Version
		return m, nil
	case diffMsg:
		// A diff that arrives for a task the reader has since left is
		// stale, and dropping it is the whole guard: openKey has already
		// pointed m.detail at the new id and asked git about that one, so
		// writing this text in would put one task's changes under another
		// task's heading and nothing on screen would say so.
		if msg.ID != m.detail {
			return m, nil
		}

		m.diff, m.worktree = msg.Text, msg.Tree
		m.diffErr, m.diffKnown, m.diffNoBase = msg.Err, true, msg.NoBase
		m.diffBase, m.diffAsking = msg.Base, false

		return m.syncPanes(), nil
	case logMsg:
		// The same guard, for the same reason: a record that arrives for a
		// task the reader has since left would put one task's history under
		// another task's heading, and nothing on screen would say so.
		if msg.ID != m.detail {
			return m, nil
		}

		m.entries, m.logErr = msg.Entries, msg.Err

		return m.syncPanes(), nil
	case filesMsg:
		// The same guard again: a listing that arrives for a task the reader
		// has since left would put one task's files under another's heading.
		if msg.ID != m.detail {
			return m, nil
		}

		m.files, m.filesErr, m.filesKnown = msg.Files, msg.Err, true

		return m.syncPanes(), nil
	case fileTextMsg:
		if msg.ID != m.detail {
			return m, nil
		}

		return m.readFile(msg), nil
	case languageMsg:
		return m.language(msg.Lang), nil
	case tea.PasteMsg:
		return m.paste(msg.Content), nil
	case tea.KeyPressMsg:
		return m.key(msg)
	case tea.MouseMsg:
		// One case for all four pointer messages, which is what the
		// interface is for. Which of them this is, and what it means, is
		// mouse.go's question.
		return m.mouse(msg)
	}

	return m, nil
}
