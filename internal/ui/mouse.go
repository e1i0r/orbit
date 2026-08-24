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
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/layout"
)

// wheelRows is how far one notch of the wheel moves the list. Three is what
// a terminal's own scrollback moves and what every list in every other
// program moves, and a list that moved a different amount would be the one
// thing on screen the reader's hand has to learn.
const wheelRows = 3

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
		return m.flip(t.Field), nil
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
func (m Model) flip(field string) Model {
	on := m.autopilotOn()
	switch {
	case field == fieldFlow:
		return m.cycleFlow()
	case field == fieldAutopilotOn && !on, field == fieldAutopilotOff && on:
		return m.autopilot()
	}
	return m
}

// rightClick moves the cursor to what was pointed at.
//
// The menu that belongs on the other end of this gesture — every verb for
// this task, the refused ones greyed with their reason — is its own task.
// Moving the cursor is the half that is right either way: the menu, when it
// arrives, is the menu for the row under the cursor.
func (m Model) rightClick(t Target) (tea.Model, tea.Cmd) {
	i, ok := m.rowOf(t)
	if !ok {
		return m, nil
	}
	return m.moveTo(i), nil
}

// wheel scrolls whatever is under it, and only over the body.
//
// On the board it moves the cursor rather than the offset, because the
// offset is not the caller's to set: follow owns it, and it is what keeps
// the cursor on screen and the last page from scrolling past its end. In a
// pane it goes through the same scroll the arrow keys do, which is the one
// site the follow rule lives at — so a reader who wheels up a live log stops
// following it, exactly as a reader who presses ↑ does.
//
// Three notches of one row rather than one of three, so that a wheel and a
// held arrow key are the same gesture as far as everything downstream is
// concerned.
func (m Model) wheel(e tea.Mouse) Model {
	if m.frame.At(e.Y) != layout.RegionBody {
		return m
	}
	up := e.Button == tea.MouseWheelUp
	if !up && e.Button != tea.MouseWheelDown {
		// The wheel pushes sideways too, and nothing here scrolls that
		// way. The pane that can — a diff wider than the terminal — is
		// scrolled with ←→, and a sideways wheel is a gesture few mice
		// have and fewer readers expect.
		return m
	}
	if m.screen == screenDetail {
		if m.hit(e.X, e.Y).Kind != TargetPaneBody {
			return m
		}
		k := firstKey(m.keys.Down)
		if up {
			k = firstKey(m.keys.Up)
		}
		for range wheelRows {
			m = m.scroll(k)
		}
		return m
	}
	if m.screen != screenList {
		return m
	}
	if up {
		return m.move(-wheelRows)
	}
	return m.move(wheelRows)
}

// firstKey is the keystroke a binding is reached by, which is the one a
// clicked hint sends and the one the wheel sends. A binding with no keys at
// all sends nothing rather than an empty keystroke every map would ignore
// anyway — the difference is that this one says so.
func firstKey(b key.Binding) keystroke {
	if keys := b.Keys(); len(keys) > 0 {
		return keystroke(keys[0])
	}
	return ""
}

// rowOf is which line of the body a target is on, by what it is rather than
// by where it was: the board is re-read twice a second, so the row a press
// landed on may have moved by the time the button comes up.
func (m Model) rowOf(t Target) (int, bool) {
	for i, r := range m.rows() {
		switch {
		case r.blank:
		case r.head && t.Kind == TargetBandHeader && r.band == t.Band:
			return i, true
		case !r.head && t.Kind == TargetTask && r.task.ID == t.ID:
			return i, true
		}
	}
	return 0, false
}

// same is whether two targets are the same thing, disregarding which field
// of it was pointed at.
func (t Target) same(o Target) bool {
	t.Column, o.Column = 0, 0
	return t == o
}
