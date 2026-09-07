package ui

// The pointer. Which cell a button went down on, which cell it came up on,
// and what that means — for both buttons, and for the wheel.
//
// Everything here goes through hit for the geometry and through the same
// methods the keyboard uses for the verbs. There is no gesture in this file
// that a key cannot also do, and that is the rule the file is written to
// keep: the pointer is a second way to reach the window's verbs, never a
// second set of them.

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/patch"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// hold is the button that is down and the cell it went down on.
//
// Holding it is what makes the gesture cancellable. A window that acts on
// the press has no way for a reader to change their mind; a window that acts
// on the release, on the target the press landed on, has the one escape
// every reader already knows — drag off the thing and let go.
type hold struct {
	target point.Target
	button tea.MouseButton
	down   bool
}

// keystroke is a key named by the binding that names it, so that a hint clicked in the
// key bar can be matched against the same bindings a real keystroke is.
//
// key.Matches is generic over fmt.Stringer, which is what makes this three
// lines rather than a synthesised terminal event: the only thing it ever
// asks a keystroke is what it is called.
type keystroke string

func (k keystroke) String() string { return string(k) }

// mouse routes one pointer event.
//
// A question waiting for an answer swallows the pointer whole, the way it
// swallows the keyboard. A reader with "cancel this run?" on screen who
// clicks a row somewhere behind it has not answered the question, and a
// window that both keeps asking and acts on the click has done something
// nobody asked for while the reader was looking at the prompt.
func (m Model) mouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.confirm != confirmNone {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		e := msg.Mouse()
		m.held = hold{target: m.hit(e.X, e.Y), button: e.Button, down: true}

		// The bar is the one thing that answers the press rather than the
		// release. A scroll bar that moved when the button came up would
		// not be a scroll bar: it is held, and what it is worth is that
		// the view is under the pointer the whole time it is held.
		if e.Button == tea.MouseLeft && m.held.target.Kind == point.ScrollBar {
			return m.scrollTo(m.held.target.Pane), nil
		}

		// A caret answers the press for the same reason: a selection is
		// dragged out of the cell the button went down on, so the field
		// has to be told where that was while the pointer is still on it.
		// The release lands on the same cell when nobody dragged
		// anywhere, and putting the caret where it already is changes
		// nothing.
		if e.Button == tea.MouseLeft && m.held.target.Kind == point.ComposeCaret {
			return m.composeAim(m.held.target), nil
		}

		return m, nil
	case tea.MouseReleaseMsg:
		return m.release(msg.Mouse())
	case tea.MouseWheelMsg:
		return m.wheel(msg.Mouse()), nil
	case tea.MouseMotionMsg:
		if m.held.down && m.held.target.Kind == point.ScrollBar {
			return m.dragBar(msg.Mouse()), nil
		}

		if m.held.down && m.held.target.Kind == point.ComposeCaret {
			return m.dragCaret(msg.Mouse()), nil
		}

		// A pointer with no button down is a reader pointing at something,
		// and on the key bar that is a question: hover answers it. Nothing
		// else here is drawn differently for being under the pointer, so
		// everything else this reports is dropped.
		return m.hover(msg.Mouse()), nil
	}

	return m, nil
}

// release is the whole gesture: the button comes up, and the press it
// completes is the one that decides what happens.
//
// The press's target is what is acted on, and it is acted on only if the
// release landed on the same thing. Which field of a row the pointer was
// over is not part of "the same thing" — a press on a task's id and a
// release on its title never left the row.
func (m Model) release(e tea.Mouse) (tea.Model, tea.Cmd) {
	held := m.held

	m.held = hold{}
	if !held.down || held.target.Kind == point.None {
		return m, nil
	}

	if !m.hit(e.X, e.Y).Same(held.target) {
		return m, nil
	}

	switch held.button {
	case tea.MouseLeft:
		return m.leftClick(held.target)
	case tea.MouseRight:
		return m.rightClick(held.target)
	}

	return m, nil
}

// leftClick is the pointer's plain gesture: point at a thing, then do the
// obvious thing to it.
//
// A row that is not the cursor's takes one click to become it and a second
// to open, which is the two-step every list in every file manager has. It is
// not a double-click: a double-click is a timer, and a timer means the same
// two clicks do different things depending on how fast the reader is.
func (m Model) leftClick(t point.Target) (tea.Model, tea.Cmd) {
	switch t.Kind {
	case point.Task:
		i, ok := m.rowOf(t)
		if !ok {
			return m, nil
		}

		if i == m.cursor {
			return m.open()
		}

		return m.moveTo(i), nil
	case point.BandHeader:
		// The band folds and unfolds, which is item five of what this
		// window is for: the bands are queues, and a queue you can shut is
		// a queue you can put down. The cursor goes to the heading first,
		// so that what the keyboard does next is what the reader just
		// pointed at.
		i, ok := m.rowOf(t)
		if !ok {
			return m, nil
		}

		return m.moveTo(i).expand(t.Band).clampCursor(), nil
	case point.BarHint:
		if t.Key == "" {
			return m, nil
		}

		return m.sendKey(keystroke(t.Key))
	case point.HeaderField:
		if t.Field == "orbit" {
			m.queueFilter = nil
			m.repoFilter = ""
			m.filter = ""

			return m.moveTo(0).clampCursor(), nil
		}

		if t.Field == "lang" {
			nextLang := "en"
			if m.opts.Words.T("header.lang_badge", "EN") == "EN" {
				nextLang = "es"
			}

			return m.applySetting("language", nextLang)
		}

		if t.Field == "repos" {
			return m.openRepos(), nil
		}

		if t.Field == "knowledge" {
			return m.openKnowledge(), nil
		}

		if t.Field == "engine" {
			return m.openEngines(), nil
		}

		if t.Field == "quota" {
			return m.openQuota(), nil
		}
	case point.StatusField:
		if t.Field == "autopilot" {
			return m.autopilot()
		}

		if t.Field == "engine" {
			return m.openEngines(), nil
		}
	case point.HeaderQueue:
		if m.queueFilter != nil && *m.queueFilter == t.Band {
			m.queueFilter = nil
			return m.moveTo(0).clampCursor(), nil
		}

		band := t.Band
		m.queueFilter = &band
		m.expanded[band] = true

		return m.jumpToBand(band)
	case point.SettingsRow:
		m.settings = m.settings.Point(t.Pane)

		rows := m.settingRowsList()
		if t.Pane >= 0 && t.Pane < len(rows) {
			r := rows[t.Pane]
			if t.Field != "" && slices.Contains(r.Options, t.Field) {
				return m.applySetting(r.Key, t.Field)
			}
		}

		return m.cycleSetting(1)
	case point.EngineRow:
		return m.chooseEngineRow(t.Pane)
	case point.PaneTab:
		return m.showTab(tab(t.Pane)), nil
	case point.Fold:
		return m.fold(t.Key), nil
	case point.Seam:
		return m.foldAttempt(t.Pane), nil
	case point.PaneRow:
		return m.openPaneRow(t.Pane)
	case point.DiffSelectToggle:
		m.diffFilePicker = !m.diffFilePicker
		if m.diffFilePicker {
			raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")
			files := patch.Files(raw)
			m.diffFileCursor = fileIndexAtOffset(files, m.panes[tabDiff].YOffset())
		}

		return m, nil
	case point.DiffFile:
		raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")

		files := patch.Files(raw)
		if t.Pane >= 0 && t.Pane < len(files) {
			m.panes[tabDiff].SetYOffset(files[t.Pane].StartLine)
			m.diffFilePicker = false
			m = m.syncPanes()
		}

		return m, nil
	case point.DialogSwitch:
		return m.flip(t.Field)
	case point.ComposeTab, point.ComposeFlowChoice,
		point.ComposeNewFlow, point.ComposeInspectFlow,
		point.ComposeField, point.ComposeCaret, point.ComposeAction, point.ComposePaste:
		return m.handleComposeClick(t)
	case point.Command:
		return m.chooseCommand(t.Key)
	case point.MenuEntry:
		// The palette's two-step again, on the menu's list. The entry is
		// found by what identifies it — glyph for a verb, name for a
		// command — never by where it sat when the button went down: the
		// list is recomputed between press and release.
		for i, e := range m.menuEntries() {
			id := e.glyph
			if id == "" && e.cmd != nil {
				id = e.cmd.Name
			}

			if id != t.Key {
				continue
			}

			if i == m.menu.sel {
				return m.chooseMenu()
			}

			next := m
			next.menu.sel = i

			return next, nil
		}

		return m, nil
	case point.FlowItem:
		return m.handleFlowClick(t)
	case point.Repo:
		return m.chooseRepo(t.ID)
	}

	return m, nil
}
