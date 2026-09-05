package ui

// The pieces one message and the input line are drawn out of.

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

// drawSupervisorInput is the line you type into, and the keys under it.
//
// It is a prompt and a dim line rather than a framed textarea: the caret
// says where the typing goes, and the screen keeps the shape every other
// screen in the cockpit has.
func (m Model) drawSupervisorInput(cw int) []string {
	p := m.opts.Words

	// The mode owns the foot of the screen. While a line is being picked
	// there is nothing to type, so it says what the keys do instead of
	// pretending.
	if m.supervisor.picking {
		return []string{
			Paint(Dim).Render(p.T("supervisor.picking_note", "the line stays in the thread, marked; the supervisor stops being told it")),
			"",
			Paint(Dim).Render(p.T("supervisor.picking_ways", "[↑↓] pick · [↵] take it back · [esc] cancel")),
		}
	}

	ways := p.T("supervisor.ways_out", "[Shift+↵] newline · [↵] send · [esc] back · [↑↓] scroll · [^R] retract · [^V] paste",
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key))

	return append(m.inputLines(cw), "", Paint(Dim).Render(ways))
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

	// The mark says "this is where you are writing", and that is true once.
	// One on every row read as three prompts — three separate things about
	// to be sent — for what is one message. The rows after the first line up
	// under it instead, so the message reads as a block.
	var rows []string

	// Two cells, which is what "❯ " is: the continuation lines start where
	// the first line's text does.
	const indent = "  "

	for _, raw := range plainLines(m.supervisor.input) {
		wrapped := splitIntoLines(raw, max(cw-2, 8))
		if len(wrapped) == 0 {
			wrapped = []string{""}
		}

		for _, l := range wrapped {
			lead := indent
			if len(rows) == 0 {
				lead = prompt
			}

			rows = append(rows, lead+l)
		}
	}

	rows[len(rows)-1] += Paint(Accent).Render("█")
	for len(rows) < 2 {
		rows = append(rows, "")
	}

	return rows
}
