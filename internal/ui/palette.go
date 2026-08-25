package ui

// The palette: the ':' line that reaches every command nobody gave a key
// to. One line at the bottom, the commands that match what has been typed
// above it, and a selection among them.
//
// It is a file of its own because it answers one question — "which command
// did the reader mean?" — and because the answer needs four small pieces
// that belong together: the text being typed, the list it filters, the
// selection inside it, and the geometry both the renderer and the pointer
// read. What a chosen command then does is not decided here: running one
// raises commandMsg and internal/cli answers it, exactly as a key verb is
// answered by whatever Control names.
//
// The input is not bubbles/textinput, for the reason ui.go gives about the
// filter: that component imports a clipboard module this build does not
// have. Like the filter, it is a plain string with a rune appended and a
// rune removed — the day it needs selection is the day the dependency is
// worth arguing for.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// paletteState is the palette while it is up, and nothing while it is down:
// open false means every other field is dead weight the next openPalette
// resets anyway.
//
// offset is the first candidate the body is showing, moved only by
// ensureVisible — the list scrolls to keep the selection on screen rather
// than the reader scrolling to find it, because the list is short enough
// that finding it was never the problem.
type paletteState struct {
	open bool

	// typed is what has been typed into the line, as a plain string.
	typed string

	sel    int // an index into candidates()
	offset int // the first candidate the body is showing
}

// openPalette brings the line up empty. Whatever was typed last time is
// last time's question; a palette that reopened mid-word would answer a
// command the reader is no longer asking.
func (m Model) openPalette() Model {
	m.palette = paletteState{open: true}
	return m
}

// closePalette takes the line down and gives the keyboard back.
func (m Model) closePalette() Model {
	m.palette = paletteState{}
	return m
}

func matchesSettingsAlias(prefix string) bool {
	for _, alias := range []string{"configuraciones", "config", "set", "ajustes"} {
		if strings.HasPrefix(alias, prefix) {
			return true
		}
	}
	return false
}

// candidates is every command whose name starts with what has been typed,
// in the order the table handed them over — the table's order is the one a
// reader learned, and reshuffling it under a prefix would move rows between
// two keystrokes.
//
// The match ignores case, as the board's filter does: a reader who types a
// capital because a sentence started with one still means the command.
func (p paletteState) candidates(cmds []Command) []Command {
	prefix := strings.ToLower(p.typed)
	var out []Command
	for _, c := range cmds {
		name := strings.ToLower(c.Name)
		if strings.HasPrefix(name, prefix) {
			out = append(out, c)
		} else if c.Name == "settings" && matchesSettingsAlias(prefix) {
			out = append(out, c)
		}
	}
	return out
}

// selected is the candidate the selection is on, or nil when there is
// nothing to be on — an empty list, or an index that lost its row to a
// shorter list after another rune landed.
func (p paletteState) selected(cmds []Command) (Command, bool) {
	all := p.candidates(cmds)
	if p.sel < 0 || p.sel >= len(all) {
		return Command{}, false
	}
	return all[p.sel], true
}

// paletteKey feeds the line, which owns every key it is not given a reason
// to give up.
//
// ↑↓ are matched against what the terminal sent rather than against the
// key map, on purpose. Up and Down carry k and j as alternates, and both
// are letters a command's name can need typed into this very line; a match
// through the map would move the selection every time the reader typed one.
func (m Model) paletteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		return m.closePalette(), nil
	case key.Matches(msg, m.keys.Open):
		return m.runSelected()
	case msg.String() == "tab":
		return m.complete(), nil
	case msg.String() == "up":
		return m.pick(-1), nil
	case msg.String() == "down":
		return m.pick(1), nil
	case msg.Code == tea.KeyBackspace:
		m.palette.typed = trimLastRune(m.palette.typed)
		return m.reselect(), nil
	}
	if msg.Text != "" {
		m.palette.typed += msg.Text
		return m.reselect(), nil
	}
	return m, nil
}

// reselect puts the selection back at the top of whatever the line now
// matches. The alternative — keeping the old index — points at a row the
// reader has never seen choose, and ⏎ would then run a command nobody
// asked for.
func (m Model) reselect() Model {
	m.palette.sel, m.palette.offset = 0, 0
	return m
}

// pick moves the selection one row, and the window with it.
func (m Model) pick(d int) Model {
	if n := len(m.palette.candidates(m.opts.Commands)); n > 0 {
		m.palette.sel += d
		if m.palette.sel < 0 {
			m.palette.sel = 0
		}
		if m.palette.sel >= n {
			m.palette.sel = n - 1
		}
	}
	return m.ensureVisible()
}

// ensureVisible moves the offset as little as it can while keeping the
// selection on screen. It reads the frame's body height, which is what the
// renderer counts rows with — one number, two readers, the rule target.go
// exists to keep.
func (m Model) ensureVisible() Model {
	h := m.frame.Body.H
	switch {
	case h <= 0:
		m.palette.offset = 0
	case m.palette.sel < m.palette.offset:
		m.palette.offset = m.palette.sel
	case m.palette.sel >= m.palette.offset+h:
		m.palette.offset = m.palette.sel - h + 1
	}
	if m.palette.offset < 0 {
		m.palette.offset = 0
	}
	return m
}

// complete fills the line with the selection's name. The list is already
// prefix-filtered, so the selection always completes what was typed — tab
// never jumps sideways to a command the reader did not start spelling.
//
// The selection goes back to the top with the line, because completing has
// made the list shorter and an index past its end would leave ⏎ pointing at
// nothing.
func (m Model) complete() Model {
	if c, ok := m.palette.selected(m.opts.Commands); ok {
		m.palette.typed = c.Name
		return m.reselect()
	}
	return m
}

// commandIndex is where a named command sits in the filtered list, which is
// what the pointer needs and the keyboard never does: the keyboard moves by
// rows, the pointer arrives holding a name.
func (m Model) commandIndex(name string) (int, bool) {
	for i, c := range m.palette.candidates(m.opts.Commands) {
		if c.Name == name {
			return i, true
		}
	}
	return 0, false
}

// paletteRows is the body while the line is up: the candidates, oldest
// table order first, cut to the rows the region has and scrolled by
// whatever ensureVisible last had to move.
func (m Model) paletteRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	all := m.palette.candidates(m.opts.Commands)
	if len(all) == 0 {
		line := Paint(Dim).Render(m.opts.Words.T("palette.none",
			"no command starts with {typed}", about("typed", m.palette.typed)))
		return fill([]string{"", fit("  "+line, w)}, h)
	}
	out := make([]string, 0, h)
	for i := m.palette.offset; i < len(all) && len(out) < h; i++ {
		out = append(out, m.paletteRow(all[i], i == m.palette.sel, w))
	}
	return fill(out, h)
}

// paletteRow draws one candidate: the name, the usage fragment when it has
// one, and either the description or — for a command the window refuses —
// the reason, on the same line, greyed.
//
// A refusal replaces the description rather than joining it, because the
// reason is the part a reader acts on and a line that carries both is a
// line that gets truncated to lose whichever mattered.
func (m Model) paletteRow(c Command, selected bool, w int) string {
	p := m.opts.Words
	tail := ""
	if c.Refused && c.Because != nil {
		tail = Paint(Dim).Render(dot + " " + c.Because(p))
	} else {
		if c.Args != "" {
			tail += Paint(Dim).Render(" " + c.Args)
		}
		if c.About != nil {
			tail += Paint(Dim).Render(dot + " " + c.About(p))
		}
	}
	line := "  " + c.Name + tail
	mark := strings.Repeat(" ", gutter)
	if selected {
		mark = markGlyph + strings.Repeat(" ", gutter-1)
		line = fit(mark+line, w)
		return Paint(Sel).Render(line)
	}
	return fit(mark+c.Name+tail, w)
}

// paletteInputLine is the line itself, where the key bar sits while the
// palette owns the keyboard. A block after the text is the caret: one cell,
// borrowed from the cursor's own paint, gone the moment the line goes down.
func (m Model) paletteInputLine(w int) string {
	if m.palette.typed == "" {
		placeholder := Paint(Dim).Render(": " + m.opts.Words.T("palette.placeholder", "type a command"))
		return fit(" "+placeholder, w)
	}
	return fit(" :"+m.palette.typed+Paint(Sel).Render(" "), w)
}

// hitPalette answers the body's cells while the palette is up: each
// candidate is a row, and the rows past the list are nothing.
//
// It walks the same offset and the same filtered list the renderer drew
// from, which is the whole deal the hit map makes: point and draw are two
// readings of one set of numbers, never two sets.
func (m Model) hitPalette(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}
	all := m.palette.candidates(m.opts.Commands)
	i := m.palette.offset + line
	if line >= m.frame.Body.H || i < 0 || i >= len(all) {
		return Target{}
	}
	return Target{Kind: TargetCommand, Key: all[i].Name}
}
