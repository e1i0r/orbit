package ui

// The menu: what can be done to the thing under the pointer, including
// what cannot, and why not.
//
// It is a file of its own because it answers one question — "what are my
// moves here?" — out of two sources that must never be allowed to drift:
// Affordances, which is what the key bar's shortlist is cut from, and the
// command table, which is what the palette reaches. The menu shows both
// lists whole; the bar and the line each show their own half.
//
// No entry has behaviour of its own. Choosing one either sends the
// keystroke its binding names — through the same map, so the refusal and
// its reason come back exactly as they would from the keyboard — or raises
// commandMsg for the command it names, through the same watch every other
// palette run uses.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// menuState is the menu while it is up. taskID is the task the menu is
// about, and empty means the board's menu — the commands that are not
// about any one task, read from the table.
type menuState struct {
	open   bool
	taskID string
	sel    int
	offset int // the first entry drawn, for a menu longer than the window
}

// openMenu brings the menu up on a target, its cursor on the first entry
// there is to choose — which is not the first row, on a menu that names its
// blocks.
func (m Model) openMenu(id string) Model {
	m.menu = menuState{open: true, taskID: id}
	m.menu.sel = max(0, menuChoice(m.menuEntries(), 0, 1))

	return m.keepMenuEntrySeen()
}

// closeMenu takes it down.
func (m Model) closeMenu() Model {
	m.menu = menuState{}
	return m
}

// openMenuForContext is m: the menu for the row under the cursor, for the
// task being viewed one level down, or — with no cursor at all — the
// board's own menu.
func (m Model) openMenuForContext() Model {
	if m.screen == screenDetail && m.detail != "" {
		return m.openMenu(m.detail)
	}

	if r, ok := m.selected(); ok && !r.head {
		return m.openMenu(r.task.ID)
	}

	return m.openMenu("")
}

// menuEntry is one drawn row of the menu: the glyph or name, the sentence
// describing it, and — when it cannot be done here — the reason on the
// same line. Exactly one of aff and cmd is set; it is what choosing does.
type menuEntry struct {
	glyph  string // the keystroke the entry sends, when it has one
	title  string // what the entry is called
	detail string // its description, dimmed, when it has one
	dim    bool   // refused here
	reason string // why, when dim
	head   bool   // names the block below it; not a thing to choose

	aff  *Affordance
	cmd  *Command
	args []string // what cmd is run with, for a command the entry can answer for
	says bool     // cmd takes a message: choosing opens the box to type it in
	tab  *tab
}

// menuEntries is the menu as it stands right now, recomputed from the board
// and the table rather than remembered — a menu frozen at open time would
// keep offering verbs for a run that finished while it was up.
//
// Which of the two it is comes from what the pointer was on when it opened.
// The board's carries the commands about no task in particular, refusals
// included: `top` greyed with "you are already in it" teaches the shape of
// the program. The task's carries the verbs, each with its reason.
func (m Model) menuEntries() []menuEntry {
	p := m.opts.Words
	if m.screen == screenDetail {
		return m.detailMenuEntries()
	}

	if m.menu.taskID == "" {
		out := make([]menuEntry, 0, len(m.opts.Commands))
		for i := range m.opts.Commands {
			c := &m.opts.Commands[i]
			// A verb about one task is not on this menu. This one is
			// opened on no row, so there is no task for such a verb to be
			// about: choosing it ran it bare and got its usage back.
			if c.AboutATask {
				continue
			}

			e := menuEntry{title: c.Name, cmd: c}
			if c.About != nil {
				e.detail = c.About(p)
			}

			if c.Refused && c.Because != nil {
				e.dim = true
				e.detail = ""
				e.reason = c.Because(p)
			}

			out = append(out, e)
		}

		return out
	}

	// A task menu with nothing on it is one whose run left the board while
	// it was up, and the drawing says so.
	return m.taskVerbEntries(m.menu.taskID)
}

// menuKey answers the keyboard while the menu is up: pick, choose, leave.
// Every other key does nothing rather than reaching past a menu the reader
// is looking at.
func (m Model) menuKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.screen == screenDetail {
		if targetTab, ok := keyToPane(msg.String()); ok {
			m = m.showTab(targetTab)
			return m.closeMenu(), nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Back):
		return m.closeMenu(), nil
	case key.Matches(msg, m.keys.Open):
		return m.chooseMenu()
	case msg.String() == "up" || key.Matches(msg, m.keys.Up):
		return m.menuPick(-1), nil
	case msg.String() == "down" || key.Matches(msg, m.keys.Down):
		return m.menuPick(1), nil
	}

	return m, nil
}

// menuPick moves the selection within the entries there are.
func (m Model) menuPick(d int) Model {
	n := len(m.menuEntries())
	if n == 0 {
		m.menu.sel = 0
		return m
	}

	es := m.menuEntries()

	// The step is a distance — the wheel moves several rows at once — and
	// the search for a row that can be chosen is one entry at a time in the
	// direction it was going.
	step := 1
	if d < 0 {
		step = -1
	}

	next := min(max(m.menu.sel+d, 0), n-1)
	if at := menuChoice(es, next, step); at >= 0 {
		next = at
	}

	m.menu.sel = next

	return m.keepMenuEntrySeen()
}

// chooseMenu acts on the selection by becoming the keyboard: an affordance
// sends the keystroke its binding names, through sendKey, so a refused verb
// refuses here with the same sentence it refuses everywhere else; a
// command runs through launch, so its output is watched like any other.
func (m Model) chooseMenu() (tea.Model, tea.Cmd) {
	entries := m.menuEntries()
	if m.menu.sel < 0 || m.menu.sel >= len(entries) {
		return m, nil
	}

	e := entries[m.menu.sel]
	if e.head {
		return m, nil
	}

	if e.tab != nil {
		m = m.showTab(*e.tab)
		return m.closeMenu(), nil
	}

	// Which task the menu is about, taken before it is closed: a command
	// that needs a message is handed the box rather than run, and the box
	// has to know what it will be talking to.
	id := m.menu.taskID

	next := m.closeMenu()

	if e.cmd != nil {
		if e.says {
			return next.openMessage(e.cmd.Name, id), nil
		}

		return next.launch(*e.cmd, e.args)
	}

	return next.sendKey(keystroke(e.glyph))
}

// menuTitleRows is how far down the first entry is drawn: the title and the
// blank under it. menuRows writes them and hitMenu counts past them, and a
// click landing one row off is what happens when only one of the two knows.
const menuTitleRows = 2

// menuRows draws the menu in the body: a line saying which menu this is,
// then one row per entry, the reason on the line for anything greyed,
// nothing hidden.
func (m Model) menuRows(h, w int) []string {
	if h <= 0 {
		return nil
	}

	p := m.opts.Words

	es := m.menuEntries()
	if len(es) == 0 {
		return fill([]string{"", fit("  "+Paint(Dim).Render(
			p.T("menu.gone", "the task this menu was opened on is no longer on the board")), w)}, h)
	}

	out := make([]string, 0, h)
	out = append(out, fit("  "+Paint(Dim).Render(m.menuTitle()), w), "")

	off := m.menuOffset(len(es), menuView(h))
	for i, e := range es[off:] {
		if len(out) >= h {
			break
		}

		out = append(out, m.menuRow(e, off+i == m.menu.sel, w))
	}

	return fill(out, h)
}

// menuTitle names the menu that is up: the two open on the same keystroke
// and answer different questions, and a reader who meant the task's and got
// the board's should not have to work that out from a missing verb.
func (m Model) menuTitle() string {
	p := m.opts.Words

	switch {
	case m.screen == screenDetail:
		return p.T("menu.title_detail", "{id} — where to look, and what can be done",
			about("id", m.menu.taskID))
	case m.menu.taskID == "":
		return p.T("menu.title_board", "commands — nothing here is about one task")
	}

	return p.T("menu.title_task", "{id} — what can be done to this task",
		about("id", m.menu.taskID))
}

// menuRow lays one entry out: glyph, name, description — or, where it
// cannot be done here, the reason instead of the description. The pieces
// mirror paletteRow's, because a reader who learned to read one list should
// be able to read the other.
func (m Model) menuRow(e menuEntry, selected bool, w int) string {
	if e.head {
		return menuHeadRow(e, w)
	}

	line := e.title
	if e.glyph != "" {
		line = Paint(Accent).Render(e.glyph) + "  " + line
	} else {
		line = "   " + line
	}

	switch {
	case e.reason != "":
		line += dot + Paint(Dim).Render(" "+e.reason)
	case e.detail != "":
		line += dot + Paint(Dim).Render(" "+e.detail)
	}

	mark := strings.Repeat(" ", gutter)
	if selected {
		mark = markGlyph + strings.Repeat(" ", gutter-1)
		return Paint(Sel).Render(fit(mark+line, w))
	}

	if e.dim {
		return fit(mark+Paint(Dim).Render(line), w)
	}

	return fit(mark+line, w)
}

// hitMenu answers the body while the menu is up: each entry is a row, and
// the rows past the list are nothing.
//
// The target carries what identifies the entry — its glyph for a verb, its
// name for a command — because the list is recomputed between press and
// release and an index could point at a different row by then.
func (m Model) hitMenu(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}

	// Past the title, which is drawn above the entries and is not one.
	line -= menuTitleRows
	if line < 0 {
		return Target{}
	}

	// Down by whatever the list has scrolled: the drawing and the counting
	// read the same offset, or a click lands on the row the reader is not
	// looking at.
	es := m.menuEntries()

	line += m.menuOffset(len(es), menuView(m.frame.Body.H))
	if line >= len(es) {
		return Target{}
	}

	e := es[line]
	if e.head {
		return Target{}
	}

	if e.glyph != "" {
		return Target{Kind: TargetMenuEntry, Key: e.glyph}
	}

	if e.cmd != nil {
		return Target{Kind: TargetMenuEntry, Key: e.cmd.Name}
	}

	return Target{}
}
