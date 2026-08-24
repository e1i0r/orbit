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
	tea "charm.land/bubbletea/v2"
)

// hold is the button that is down and the cell it went down on.
//
// Holding it is what makes the gesture cancellable. A window that acts on
// the press has no way for a reader to change their mind; a window that acts
// on the release, on the target the press landed on, has the one escape
// every reader already knows — drag off the thing and let go.
type hold struct {
	target Target
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
		return m, nil
	case tea.MouseReleaseMsg:
		return m.release(msg.Mouse())
	case tea.MouseWheelMsg:
		return m.wheel(msg.Mouse()), nil
	case tea.MouseMotionMsg:
		// Ignored, and ignored here rather than by leaving the case out.
		// The window asks for MouseModeCellMotion, so a drag reports a
		// message per cell it crosses; the release carries the cell it
		// happened in, which is everything this window needs, and nothing
		// is drawn differently while a button is held. When something is —
		// a hovered row, a drag that reorders — this is where it starts.
		return m, nil
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
	if !held.down || held.target.Kind == TargetNone {
		return m, nil
	}
	if !m.hit(e.X, e.Y).same(held.target) {
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
func (m Model) leftClick(t Target) (tea.Model, tea.Cmd) {
	switch t.Kind {
	case TargetTask:
		i, ok := m.rowOf(t)
		if !ok {
			return m, nil
		}
		if i == m.cursor {
			return m.open()
		}
		return m.moveTo(i), nil
	case TargetBandHeader:
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
	case TargetBarHint:
		if t.Key == "" {
			return m, nil
		}
		return m.sendKey(keystroke(t.Key))
	case TargetPaneTab:
		return m.showTab(tab(t.Pane)), nil
	case TargetDialogSwitch:
		return m.flip(t.Field)
	case TargetDialogPhase:
		return m.openEngines(), nil
	case TargetCommand:
		// The same two-step a task row takes: the first click selects,
		// the second runs what was selected. Both arrive through the same
		// methods the keyboard uses — there is no third path to a run.
		i, ok := m.commandIndex(t.Key)
		if !ok {
			return m, nil
		}
		if i == m.palette.sel {
			return m.runSelected()
		}
		next := m
		next.palette.sel = i
		return next.ensureVisible(), nil
	case TargetMenuEntry:
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
	}
	// TargetPaneBody and TargetDialogPhase are pointed at and not acted on.
	// The pane is already where the keyboard is, so a click in it has
	// nothing to change; a phase becomes something to click when there is
	// something to change about it.
	return m, nil
}

// sendKey puts a keystroke through the same map a pressed key goes through,
// which is how a clicked hint reaches the verb it names.
//
// It is the screen's map and not always the board's, because the key bar is
// drawn on all three screens and says something different on each. The
// filter is the one place a click cannot go: it is text being typed, and a
// pointer has nothing to type.
func (m Model) sendKey(k keystroke) (tea.Model, tea.Cmd) {
	switch {
	case m.filtering:
		return m, nil
	case m.screen == screenStart:
		return m.startKey(k)
	case m.screen == screenDetail:
		return m.detailKey(k)
	}
	return m.listKey(k)
}

// flip is one of the start dialog's switches, clicked.
//
// The two positions of the autopilot switch are two rows, and clicking the
// one that is already chosen does nothing. That is the difference between a
// switch and a button: a reader who clicks "on" means on, not "the other
// one", and a row that toggled whichever way it was pointed at would turn
// autopilot off for the reader who clicked the word on.
func (m Model) flip(field string) (tea.Model, tea.Cmd) {
	on := m.autopilotOn()
	switch {
	case field == fieldFlow:
		return m.cycleFlow(), nil
	case field == fieldAutopilotOn && !on, field == fieldAutopilotOff && on:
		return m.autopilot()
	}
	return m, nil
}

// rightClick opens the menu for what was pointed at.
//
// The cursor goes to the row first, so that what happens next is not a
// surprise: the menu is the one the keyboard's m would have opened on that
// row, and every key pressed afterwards acts where the reader just looked.
// In the task view there is no row to move to — the pane is the task — so
// the menu simply opens on the task being viewed.
func (m Model) rightClick(t Target) (tea.Model, tea.Cmd) {
	if t.Kind == TargetPaneBody {
		if s := m.subject(); s.ID != "" {
			return m.openMenu(s.ID), nil
		}
		return m, nil
	}
	i, ok := m.rowOf(t)
	if !ok {
		return m, nil
	}
	next := m.moveTo(i)
	if t.Kind == TargetTask {
		return next.openMenu(t.ID), nil
	}
	return next, nil
}
