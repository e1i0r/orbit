package ui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// supervisorRows draws the whole screen: a heading, the thread under it,
// and what you are about to say at the foot.
//
// It is laid out the way the quota and engine screens are — a title, an
// indented body, the ways out in dim underneath — and not as two framed
// boxes. Framed, it read as a different program bolted into the cockpit.
func (m Model) supervisorRows(h, w int) []string {
	if h <= 0 {
		return nil
	}

	cw, threadH := m.supervisorLayout(h, w)

	out := m.supervisorHead(cw)
	out = append(out, m.supervisorBody(threadH, cw)...)
	out = append(out, "")
	out = append(out, m.drawCompletions(cw)...)
	out = append(out, m.drawSupervisorInput(cw)...)

	for i, row := range out {
		out[i] = fit("  "+row, w)
	}

	return fill(out, h)
}

// supervisorHeadRows is the heading's height: the blank row above the
// title, the title, the standing facts under it, and the blank row that
// sets the thread off.
const supervisorHeadRows = 4

// supervisorLayout is the screen's arithmetic, in one place: the width every
// line is wrapped to, and how tall the thread is once the heading and the
// input have taken what they need. The keys ask it the same question the
// drawing does — a scroll that clamped against a different height from the
// one on screen is a scroll that stops a row early, or does nothing for the
// first ten presses.
func (m Model) supervisorLayout(h, w int) (cw, threadH int) {
	cw = max(min(w-4, 110), 24)
	// The offers take their rows from the thread, not from the line being
	// typed: a list that pushed the input off the bottom would hide the
	// cursor it is helping somebody move.
	return cw, max(h-supervisorHeadRows-1-len(m.drawSupervisorInput(cw))-len(m.drawCompletions(cw)), 3)
}

// supervisorHead is the title and the standing facts under it: which engine
// answers here, whether autopilot is on, and how long the thread is.
func (m Model) supervisorHead(cw int) []string {
	p := m.opts.Words

	title := p.T("supervisor.title", "Supervisor & Cockpit Memory")
	if m.supervisor.picking {
		title = p.T("supervisor.picking", "pick a line to take back")
	}

	auto := p.T("supervisor.auto_off", "autopilot off")
	if m.autopilotOn() {
		auto = p.T("supervisor.auto_on", "autopilot on")
	}

	facts := strings.Join([]string{
		p.T("supervisor.answered_by", "answered by {engine}", about("engine", m.dialEngine(m.knobs.Engine))),
		auto,
		p.T("supervisor.msg_count", "{n} messages", about("n", strconv.Itoa(len(m.supervisor.lines)))),
	}, " · ")

	return []string{"", Paint(Accent).Render(title), Paint(Dim).Render(fit(facts, cw)), ""}
}

// supervisorBody is the thread's content: every message, then the window of
// it that fits, chosen by the scroll offset or by what is being picked, with
// a scroll bar down its right edge when there is more of it than fits.
//
// A thread shorter than the screen is padded above rather than below, so the
// last thing said sits against the line you answer it in. Padding underneath
// left a conversation of two lines stranded at the top of a tall terminal
// with a field of nothing between it and the cursor.
func (m Model) supervisorBody(maxRows, cw int) []string {
	rendered, starts := m.threadLines(cw)
	if len(rendered) < maxRows {
		return append(make([]string, maxRows-len(rendered)), rendered...)
	}

	offset := m.threadOffset(len(rendered), maxRows, starts)
	rows := slices.Clone(rendered[offset : offset+maxRows])

	// The rail stands in the column after the text, which is why the rows
	// are filled out to one width first: a bar drawn against ragged lines
	// zigzags down the screen instead of standing still.
	track := scrollTrack(maxRows, len(rendered), offset)
	for i := range rows {
		if track == nil {
			break
		}

		rows[i] = padRight(fit(rows[i], cw), cw) + track[i]
	}

	return rows
}

// threadLines is every message rendered, and which row each one starts on
// so that picking one can scroll to it.
//
// It renders once per change rather than once per frame: what came back last
// time is given back whenever the thread, the width and the way it is being
// read are all still what they were. supervisorcache.go says why that is
// worth doing.
func (m Model) threadLines(cw int) (rows []string, starts []int) {
	key := m.threadKeyAt(cw)
	if rows, starts, held := m.thread.rowsFor(key); held {
		return rows, starts
	}

	rows, starts = m.renderThread(cw)
	m.thread.keep(key, rows, starts)

	return rows, starts
}

// renderThread draws every message in the thread, whatever it costs.
func (m Model) renderThread(cw int) (rows []string, starts []int) {
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
	if m.supervisor.follow {
		offset = max(total-maxRows, 0)
	}

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
	if m.isEngineName(l.By) {
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
	case m.isEngineName(l.By):
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
	eng := m.dialEngine(m.knobs.Engine)

	rail := Paint(Live).Render("▎")
	head := railed(rail, Paint(Dim).Render(m.now.Format("15:04:05"))+"  "+Paint(Live).Bold(true).Render(eng)+" "+Paint(Dim).Render("[supervisor]"))
	body := railed(rail, m.spinner(Live)+Paint(Dim).Render(m.opts.Words.T("supervisor.thinking", "supervisor is thinking...")))

	return []string{fit(head, cw), fit(body, cw), ""}
}

// isEngineName is whether a line was written by a model rather than a person.
//
// The roster is asked first, so an engine added to internal/engine is
// recognised here without a second edit. This was a list of five names, and
// a sixth engine's answers were drawn in a person's colour and re-wrapped as
// plain text rather than rendered as the markdown they are.
//
// The names below it stay, and are not the roster's job: a thread is a
// record, it can hold lines written months ago by an engine this build no
// longer has, and a person did not type those either.
func (m Model) isEngineName(by string) bool {
	if by == "supervisor" || slices.Contains(m.engineNames(), by) {
		return true
	}

	switch by {
	case "claude", "codex", "opencode", "gemini":
		return true
	}

	return false
}

// plainLines is one message's text as lines, whatever wrote the newlines.
func plainLines(text string) []string {
	return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
}
