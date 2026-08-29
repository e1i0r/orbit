package ui

// The supervisor screen's frame: boxes of one width, and the pieces one
// message is drawn out of.

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// The frame is built out of strings rather than lipgloss.Border because
// every row on this screen is already a string of styled cells, assembled
// and then measured with lipgloss.Width — a border applied to a block would
// have to re-measure text that carries escape sequences, and the two
// measurements disagreeing is exactly how a frame comes out crooked.
const (
	boxTL, boxTR = "╭", "╮"
	boxBL, boxBR = "╰", "╯"
	boxH, boxV   = "─", "│"
)

// supervisorBoxWidth is the one width the whole screen is drawn to.
//
// One and not three. The rule under the title, the thread's text and the
// input box each working out their own leaves the screen a couple of cells
// crooked in a way no single line looks wrong on its own. One function,
// called by everything, is what makes the edges line up.
func supervisorBoxWidth(w int) int {
	return max(min(w-4, 110), 24)
}

// boxContentWidth is what fits between the two verticals and their padding.
func boxContentWidth(boxW int) int {
	return max(boxW-4, 1)
}

// boxTop and boxBottom are borders with a label let into the left of the
// rule and, when there is one, a second label pushed to the right of it:
//
//	╭─ 🛸 Supervisor ──────────────────────────── 17 messages ─╮
//
// Putting the title in the border rather than on a line of its own is worth
// two rows of thread on a short terminal, and it is what makes the box read
// as one thing instead of a heading with a rectangle under it.
func boxTop(role Role, label, right string, w int) string {
	return boxEdge(role, boxTL, boxTR, label, right, w)
}

func boxBottom(role Role, label, right string, w int) string {
	return boxEdge(role, boxBL, boxBR, label, right, w)
}

func boxEdge(role Role, corner, endCorner, label, right string, w int) string {
	rule := Paint(role)
	head := rule.Render(corner + boxH)
	used := 2

	if label != "" {
		head += " " + label + " "
		used += lipgloss.Width(label) + 2
	}

	tail := rule.Render(boxH + endCorner)
	tailCells := 2

	if right != "" {
		tail = " " + right + " " + tail
		tailCells += lipgloss.Width(right) + 2
	}

	return fit(head+rule.Render(strings.Repeat(boxH, max(w-used-tailCells, 1)))+tail, w)
}

// boxRow is one line inside a box, padded so the right vertical of every row
// lands in the same column.
func boxRow(role Role, content string, w int) string {
	inner := boxContentWidth(w)
	return Paint(role).Render(boxV) + " " + padRight(fit(content, inner), inner) + " " + Paint(role).Render(boxV)
}

// railed puts a message's rail in front of one line of its body.
//
// The rail is what makes a message read as one block: a paragraph of six
// wrapped lines under a name is otherwise indistinguishable from six things
// somebody said, and the thread is read by scanning down the left edge.
func railed(rail, text string) string {
	return rail + " " + text
}

// markdownIndent is the fixed indent renderMarkdown puts on every line it
// returns. The rail replaces it, exactly, so a rendered heading and a
// wrapped sentence still start in the same column.
const markdownIndent = "    "

// drawSupervisorTextarea is the box you type into.
//
// It is the same width as the thread above it and closed on all four sides,
// which the old one was not: the top and bottom rules ended two cells short
// of the rule above them and there was no right edge at all, so the screen
// read as three things that had been drawn separately.
func (m Model) drawSupervisorTextarea(boxW int) []string {
	p := m.opts.Words
	cw := boxContentWidth(boxW)

	// The mode owns the box. While a line is being picked there is nothing
	// to type, so the box says what the keys do instead of pretending.
	if m.supervisor.picking {
		ways := p.T("supervisor.picking_ways", "[↑↓] pick · [↵] take it back · [esc] cancel")

		return []string{
			boxTop(Accent, Paint(Accent).Bold(true).Render("✂ "+p.T("supervisor.picking", "pick a line to take back")), "", boxW),
			boxRow(Accent, Paint(Dim).Render(p.T("supervisor.picking_note", "the line stays in the thread, marked; the supervisor stops being told it")), boxW),
			boxBottom(Accent, Paint(Dim).Render(ways), "", boxW),
		}
	}

	title := Paint(Accent).Bold(true).Render("💬 " + p.T("supervisor.input_prompt", "Say to Supervisor / Directive"))

	rows := []string{boxTop(Accent, title, "", boxW)}
	for _, l := range m.inputLines(cw) {
		rows = append(rows, boxRow(Accent, l, boxW))
	}

	ways := p.T("supervisor.ways_out", "[Shift+↵] newline · [↵] send · [esc] back · [↑↓] scroll · [^R] retract · [^V] paste",
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key))

	return append(rows, boxBottom(Accent, Paint(Dim).Render(ways), "", boxW))
}

// inputLines is what has been typed, wrapped, with the cursor on the end of
// the last line. It is never fewer than two rows, so the box does not resize
// under the first character.
func (m Model) inputLines(cw int) []string {
	prompt := Paint(Accent).Render("❯ ")

	if m.supervisor.input == "" {
		placeholder := m.opts.Words.T("supervisor.placeholder", "type a briefing, question or standing directive...")
		return []string{prompt + Paint(Dim).Render(placeholder) + Paint(Accent).Render("█"), ""}
	}

	var rows []string

	for _, raw := range plainLines(m.supervisor.input) {
		wrapped := splitIntoLines(raw, max(cw-2, 8))
		if len(wrapped) == 0 {
			wrapped = []string{""}
		}

		for _, l := range wrapped {
			rows = append(rows, prompt+l)
		}
	}

	rows[len(rows)-1] += Paint(Accent).Render("█")
	for len(rows) < 2 {
		rows = append(rows, "")
	}

	return rows
}
