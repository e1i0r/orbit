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
}

// openMenu brings the menu up on a target.
func (m Model) openMenu(id string) Model {
	m.menu = menuState{open: true, taskID: id}
	return m
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

	aff *Affordance
	cmd *Command
	tab *tab
}

// menuEntries is the menu as it stands right now, recomputed from the board
// and the table rather than remembered — a menu frozen at open time would
// keep offering verbs for a run that finished while it was up.
//
// The board's menu carries the whole table, refusals included: `top` greyed
// with "you are already in it" teaches the shape of the program, which is
// more than hiding it would.
func (m Model) menuEntries() []menuEntry {
	p := m.opts.Words
	if m.screen == screenDetail {
		return m.tabMenuEntries()
	}

	if m.menu.taskID == "" {
		out := make([]menuEntry, 0, len(m.opts.Commands))
		for i := range m.opts.Commands {
			c := &m.opts.Commands[i]

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

	t, ok := m.task(m.menu.taskID)
	if !ok {
		// The run the menu was opened on left the board while it was up.
		return nil
	}

	all := m.keys.Affordances(t, m.conditions(t))

	out := make([]menuEntry, 0, len(all))
	for i := range all {
		a := &all[i]

		e := menuEntry{
			glyph: a.Key.Help().Key,
			title: a.Key.Help().Desc,
			aff:   a,
		}
		if !a.OK {
			e.dim = true
			e.reason = a.Why(p)
		}

		out = append(out, e)
	}

	return out
}

func (m Model) tabMenuEntries() []menuEntry {
	p := m.opts.Words
	descs := map[tab]string{
		tabOverview:  p.T("tab_desc.overview", "general status, live activity and metrics summary"),
		tabFlow:      p.T("tab_desc.flow", "task pipeline phases and execution plan"),
		tabGates:     p.T("tab_desc.gates", "automated quality gates, linters and validation status"),
		tabCost:      p.T("tab_desc.cost", "token usage and monetary cost breakdown per phase"),
		tabRefused:   p.T("tab_desc.refused", "denied tool invocations and permission rejections"),
		tabTimeline:  p.T("tab_desc.timeline", "complete live event timeline and phase history"),
		tabReport:    p.T("tab_desc.report", "final solution report, summary and review conclusions"),
		tabArtifacts: p.T("tab_desc.artifacts", "raw tool output and generated artifacts"),
		tabNotes:     p.T("tab_desc.notes", "operator notes and interactive dialogue history"),
		tabDiff:      p.T("tab_desc.diff", "git working tree diff and code modifications"),
		tabThinking:  p.T("tab_desc.thinking", "extended model thinking, chain of thought and reasoning"),
	}

	var out []menuEntry

	for _, n := range m.tabNames() {
		tVal := n.tab
		k := paneKey(tVal)
		out = append(out, menuEntry{
			glyph:  "[" + k + "]",
			title:  n.text,
			detail: descs[tVal],
			tab:    &tVal,
		})
	}

	return out
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

	m.menu.sel += d
	if m.menu.sel < 0 {
		m.menu.sel = 0
	}

	if m.menu.sel >= n {
		m.menu.sel = n - 1
	}

	return m
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
	if e.tab != nil {
		m = m.showTab(*e.tab)
		return m.closeMenu(), nil
	}

	next := m.closeMenu()

	if e.cmd != nil {
		// A command that takes the id of a task is handed to the command
		// line with its name already typed, rather than run. The menu
		// chooses with no arguments and has no id to give: running one of
		// these bare printed "requeue needs the id of a task" into the
		// watch, which is an answer to a question nobody asked and a dead
		// end — there is nowhere in the menu to say which task. The line
		// is where an id can be typed, and it shows the usage under it.
		if e.cmd.NeedsTask {
			return next.openPaletteWith(e.cmd.Name + " "), nil
		}

		return next.launch(*e.cmd, nil)
	}

	return next.sendKey(keystroke(e.glyph))
}

// menuRows draws the menu in the body: one row per entry, the reason on the
// line for anything greyed, nothing hidden.
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
	for i, e := range es {
		if len(out) >= h {
			break
		}

		out = append(out, m.menuRow(e, i == m.menu.sel, w))
	}

	return fill(out, h)
}

// menuRow lays one entry out: glyph, name, description — or, where it
// cannot be done here, the reason instead of the description. The pieces
// mirror paletteRow's, because a reader who learned to read one list should
// be able to read the other.
func (m Model) menuRow(e menuEntry, selected bool, w int) string {
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

	es := m.menuEntries()
	if line < 0 || line >= len(es) {
		return Target{}
	}

	e := es[line]
	if e.glyph != "" {
		return Target{Kind: TargetMenuEntry, Key: e.glyph}
	}

	if e.cmd != nil {
		return Target{Kind: TargetMenuEntry, Key: e.cmd.Name}
	}

	return Target{}
}
