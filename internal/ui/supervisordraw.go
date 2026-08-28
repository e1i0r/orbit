package ui

import (
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// supervisorRows draws the whole screen: the thread in one box and what you
// are about to say in another, both of one width.
//
// The two boxes are sized from each other rather than from constants. The
// input grows with what has been typed into it, and the thread is whatever
// is left, so a long directive eats the conversation instead of being
// clipped against a fixed six rows.
func (m Model) supervisorRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	boxW := supervisorBoxWidth(w)
	input := m.drawSupervisorTextarea(boxW)

	threadH := h - len(input) - 1
	if threadH < 3 {
		threadH = 3
	}

	out := m.drawSupervisorThread(threadH, boxW)
	out = append(out, "")
	out = append(out, input...)
	for i, row := range out {
		out[i] = fit("  "+row, w)
	}
	return fill(out, h)
}

// drawSupervisorThread is the conversation, framed.
func (m Model) drawSupervisorThread(h, boxW int) []string {
	p := m.opts.Words
	eng := m.knobs.Engine
	if eng == "" {
		eng = "claude"
	}
	autoPip := Paint(Dim).Render("○ off")
	if m.autopilotOn() {
		autoPip = Paint(Live).Render("● on")
	}

	// The border says which mode the screen is in. Picking a line to take
	// back changes what every key does, and a mode you cannot see is a mode
	// you press keys into by accident.
	frame := Dim
	right := Paint(Dim).Render("⚡ ") + autoPip + Paint(Dim).Render(" · "+strconv.Itoa(len(m.supervisor.lines))+" msg")
	if m.supervisor.picking {
		frame = Accent
		right = Paint(Accent).Bold(true).Render(p.T("supervisor.picking", "pick a line to take back"))
	}

	title := Paint(Accent).Bold(true).Render("🛸 "+p.T("supervisor.title", "Supervisor & Cockpit Memory")) +
		" " + Paint(Dim).Render("·") + " " + Paint(Accent).Render("["+eng+"]")

	rows := []string{boxTop(frame, title, right, boxW)}
	for _, body := range m.supervisorBody(h-2, boxContentWidth(boxW)) {
		rows = append(rows, boxRow(frame, body, boxW))
	}
	return append(rows, boxBottom(frame, "", "", boxW))
}

// supervisorBody is the thread's content: every message, then the window of
// it that fits, chosen by the scroll offset or by what is being picked.
//
// A thread shorter than the box is padded above rather than below, so the
// last thing said sits against the box you answer it in. Padding underneath
// left a conversation of two lines stranded at the top of a tall terminal
// with a field of nothing between it and the cursor.
func (m Model) supervisorBody(maxRows, cw int) []string {
	rendered, starts := m.threadLines(cw)
	if len(rendered) < maxRows {
		return append(make([]string, maxRows-len(rendered)), rendered...)
	}
	offset := m.threadOffset(len(rendered), maxRows, starts)
	return rendered[offset : offset+maxRows]
}

// threadLines renders every message, and says which row each one starts on
// so that picking one can scroll to it.
func (m Model) threadLines(cw int) (rows []string, starts []int) {
	p := m.opts.Words
	if len(m.supervisor.lines) == 0 && !m.supervisorBusy {
		empty := p.T("supervisor.empty", "No messages in supervisor thread yet. Type a briefing or instruction below.")
		return append([]string{""}, splitIntoLines(Paint(Dim).Render(empty), cw)...), nil
	}
	for i, l := range m.supervisor.lines {
		starts = append(starts, len(rows))
		rows = append(rows, m.messageLines(l, cw, m.supervisor.picking && m.supervisor.pick == i)...)
		rows = append(rows, "")
	}
	if m.supervisorBusy {
		rows = append(rows, m.supervisorThinking(cw)...)
	}
	return rows, starts
}

// threadOffset is which row the window starts at.
//
// Scrolling and picking are the same movement seen from two sides: when a
// line is being picked the offset is whatever keeps it on screen, and the
// scroll position is not something the reader has to manage as well.
func (m Model) threadOffset(total, maxRows int, starts []int) int {
	offset := m.supervisor.offset
	if m.supervisor.picking && m.supervisor.pick < len(starts) {
		start := starts[m.supervisor.pick]
		end := total
		if m.supervisor.pick+1 < len(starts) {
			end = starts[m.supervisor.pick+1]
		}
		switch {
		case start < offset:
			offset = start
		case end > offset+maxRows:
			offset = end - maxRows
		}
	}
	return min(max(offset, 0), max(total-maxRows, 0))
}

// messageLines is one message: who said it, and what they said under a rail
// in their colour.
func (m Model) messageLines(l view.SupervisorLine, cw int, selected bool) []string {
	p := m.opts.Words
	role := Accent
	if isEngineName(l.By) {
		role = Live
	}
	if l.Retracted {
		role = Dim
	}
	rail := Paint(role).Render("▎")
	if selected {
		rail = Paint(Accent).Bold(true).Render("▶")
	}

	who := Paint(role).Bold(true).Render(l.By) + " " + Paint(Dim).Render("["+l.Channel+"]")
	tag := ""
	if l.TaskID != "" {
		tag = Paint(Accent).Render("(" + l.TaskID + ")")
	}
	if l.Retracted {
		tag = strings.TrimSpace(tag + " " + Paint(Dim).Render(p.T("supervisor.retracted", "(retracted)")))
	}
	if selected {
		tag = Paint(Accent).Bold(true).Render(p.T("supervisor.take_back", "[↵] take this one back"))
	}

	head := railed(rail, Paint(Dim).Render(l.At.Format("15:04:05"))+"  "+who)
	rows := []string{spread(head, tag, cw)}
	for _, body := range m.messageBody(l, cw) {
		rows = append(rows, railed(rail, body))
	}
	return rows
}

// messageBody is what was said, wrapped to the rail's column.
//
// Only the engine's own replies are read as Markdown. What an operator types
// is a sentence, and a sentence that happens to start with a dash is not a
// bullet.
func (m Model) messageBody(l view.SupervisorLine, cw int) []string {
	text := max(cw-2, 8)
	var out []string
	switch {
	case l.Retracted:
		for _, raw := range plainLines(l.Text) {
			for _, wrapped := range splitIntoLines(raw, text) {
				out = append(out, Paint(Dim).Render(wrapped))
			}
		}
	case isEngineName(l.By):
		for _, md := range renderMarkdown(l.Text, text, false) {
			out = append(out, strings.TrimPrefix(md, markdownIndent))
		}
	default:
		for _, raw := range plainLines(l.Text) {
			out = append(out, splitIntoLines(formatInlineMarkdown(raw), text)...)
		}
	}
	return out
}

// supervisorThinking is the engine's turn before it has said anything.
func (m Model) supervisorThinking(cw int) []string {
	eng := m.knobs.Engine
	if eng == "" {
		eng = "claude"
	}
	rail := Paint(Live).Render("▎")
	spin := spinnerFrames[int(m.now.UnixMilli()/120)%len(spinnerFrames)]
	head := railed(rail, Paint(Dim).Render(m.now.Format("15:04:05"))+"  "+Paint(Live).Bold(true).Render(eng)+" "+Paint(Dim).Render("[supervisor]"))
	body := railed(rail, Paint(Live).Render(spin+" ")+Paint(Dim).Render(m.opts.Words.T("supervisor.thinking", "supervisor is thinking...")))
	return []string{fit(head, cw), fit(body, cw), ""}
}

// isEngineName is whether a line was written by a model rather than a person.
func isEngineName(by string) bool {
	switch by {
	case "supervisor", "claude", "codex", "opencode", "gemini":
		return true
	}
	return false
}

// plainLines is one message's text as lines, whatever wrote the newlines.
func plainLines(text string) []string {
	return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
}
