package supervisor

import (
	"slices"
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/markdown"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// supervisorRows draws the whole screen: a heading, the thread under it,
// and what you are about to say at the foot.
//
// It is laid out the way the quota and engine screens are — a title, an
// indented body, the ways out in dim underneath — and not as two framed
// boxes. Framed, it read as a different program bolted into the cockpit.
func (s State) rows(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	cw, threadH := s.layout(h, w, e)

	out := s.supervisorHead(cw, e)
	out = append(out, s.supervisorBody(threadH, cw, e)...)
	out = append(out, "")
	out = append(out, s.drawCompletions(cw, e)...)
	out = append(out, s.drawSupervisorInput(cw, e)...)

	out = besideThread(out, s.knownSide(h, w, e), cw)

	for i, row := range out {
		out[i] = cells.Fit("  "+row, w)
	}

	return cells.Fill(out, h)
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
func (s State) layout(h, w int, e Env) (cw, threadH int) {
	cw = max(min(w-4, 110), 24)
	if s.sideFits(w) {
		cw = min(cw, w-sideGap-sideWidth)
	}
	// The offers take their rows from the thread, not from the line being
	// typed: a list that pushed the input off the bottom would hide the
	// cursor it is helping somebody move.
	return cw, max(h-supervisorHeadRows-1-len(s.drawSupervisorInput(cw, e))-len(s.drawCompletions(cw, e)), 3)
}

// supervisorHead is the title and the standing facts under it: which engine
// answers here, whether autopilot is on, and how long the thread is.
func (s State) supervisorHead(cw int, e Env) []string {
	p := e.Words

	title := p.T("supervisor.title", "Supervisor & Cockpit Memory")

	switch {
	case s.picking:
		title = p.T("supervisor.picking", "pick a line to take back")
	case s.list:
		title = p.T("supervisor.conversations", "Conversations")
	}

	auto := p.T("supervisor.auto_off", "autopilot off")
	if e.Autopilot {
		auto = p.T("supervisor.auto_on", "autopilot on")
	}

	facts := strings.Join([]string{
		p.T("supervisor.answered_by", "answered by {engine}", about("engine", e.Engine)),
		auto,
		s.howMuch(e),
	}, " · ")

	return []string{"", theme.Paint(theme.Accent).Render(title), theme.Paint(theme.Dim).Render(cells.Fit(facts, cw)), ""}
}

// howMuch is how much there is: how many conversations while the list is
// up, and how long the open one is while it is being read.
func (s State) howMuch(e Env) string {
	p := e.Words

	if s.list {
		return p.P("supervisor.conv_count", len(conversationsOf(s.all)),
			"{n} conversation", "{n} conversations")
	}

	return p.P("supervisor.msg_count2", len(s.lines), "{n} message", "{n} messages")
}

// supervisorBody is the thread's content: every message, then the window of
// it that fits, chosen by the scroll offset or by what is being picked, with
// a scroll bar down its right edge when there is more of it than fits.
//
// A thread shorter than the screen is padded above rather than below, so the
// last thing said sits against the line you answer it in. Padding underneath
// left a conversation of two lines stranded at the top of a tall terminal
// with a field of nothing between it and the cursor.
func (s State) supervisorBody(maxRows, cw int, e Env) []string {
	if s.list {
		return s.conversationRows(maxRows, cw, e)
	}

	rendered, starts := s.threadLines(cw, e)
	if len(rendered) < maxRows {
		return append(make([]string, maxRows-len(rendered)), rendered...)
	}

	offset := s.threadOffset(len(rendered), maxRows, starts)
	rows := slices.Clone(rendered[offset : offset+maxRows])

	// The rail stands in the column after the text, which is why the rows
	// are filled out to one width first: a bar drawn against ragged lines
	// zigzags down the screen instead of standing still.
	track := cells.Track(maxRows, len(rendered), offset)
	for i := range rows {
		if track == nil {
			break
		}

		rows[i] = cells.PadRight(cells.Fit(rows[i], cw), cw) + track[i]
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
func (s State) threadLines(cw int, e Env) (rows []string, starts []int) {
	key := s.threadKeyAt(cw, e)
	if rows, starts, held := s.thread.rowsFor(key); held {
		return rows, starts
	}

	rows, starts = s.renderThread(cw, e)
	s.thread.keep(key, rows, starts)

	return rows, starts
}

// renderThread draws every message in the thread, whatever it costs.
func (s State) renderThread(cw int, e Env) (rows []string, starts []int) {
	p := e.Words
	if len(s.lines) == 0 && !e.Busy {
		empty := p.T("supervisor.empty", "No messages in supervisor thread yet. Type a briefing or instruction below.")
		return append([]string{""}, cells.Lines(theme.Paint(theme.Dim).Render(empty), cw)...), nil
	}

	for i, l := range s.lines {
		starts = append(starts, len(rows))
		rows = append(rows, s.messageLines(l, cw, s.picking && s.pick == i, e)...)
		rows = append(rows, "")
	}

	if e.Busy {
		rows = append(rows, s.supervisorThinking(cw, e)...)
	}

	return rows, starts
}

// threadOffset is which row the window starts at.
//
// Scrolling and picking are the same movement seen from two sides: when a
// line is being picked the offset is whatever keeps it on screen, and the
// scroll position is not something the reader has to manage as well.
func (s State) threadOffset(total, maxRows int, starts []int) int {
	offset := s.offset
	if s.follow {
		offset = max(total-maxRows, 0)
	}

	if s.picking && s.pick < len(starts) {
		start := starts[s.pick]

		end := total
		if s.pick+1 < len(starts) {
			end = starts[s.pick+1]
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
func (s State) messageLines(l view.SupervisorLine, cw int, selected bool, e Env) []string {
	p := e.Words

	role := theme.Accent
	if isEngineName(l.By, e) {
		role = theme.Live
	}

	if l.Retracted {
		role = theme.Dim
	}

	rail := theme.Paint(role).Render("▎")
	if selected {
		rail = theme.Paint(theme.Accent).Bold(true).Render("▶")
	}

	who := theme.Paint(role).Bold(true).Render(l.By) + " " + theme.Paint(theme.Dim).Render("["+l.Channel+"]")

	tag := ""
	if l.TaskID != "" {
		tag = theme.Paint(theme.Accent).Render("(" + l.TaskID + ")")
	}

	if l.Retracted {
		tag = strings.TrimSpace(tag + " " + theme.Paint(theme.Dim).Render(p.T("supervisor.retracted", "(retracted)")))
	}

	if selected {
		tag = theme.Paint(theme.Accent).Bold(true).Render(p.T("supervisor.take_back", "[↵] take this one back"))
	}

	head := railed(rail, theme.Paint(theme.Dim).Render(l.At.Format("15:04:05"))+"  "+who)

	rows := []string{cells.Spread(head, tag, cw)}
	for _, body := range s.messageBody(l, cw, e) {
		rows = append(rows, railed(rail, body))
	}

	return rows
}

// messageBody is what was said, wrapped to the rail's column.
//
// Only the engine's own replies are read as Markdown. What an operator types
// is a sentence, and a sentence that happens to start with a dash is not a
// bullet.
func (s State) messageBody(l view.SupervisorLine, cw int, e Env) []string {
	text := max(cw-2, 8)

	var out []string

	switch {
	case l.Retracted:
		for _, raw := range plainLines(l.Text) {
			for _, wrapped := range cells.Lines(raw, text) {
				out = append(out, theme.Paint(theme.Dim).Render(wrapped))
			}
		}
	case isEngineName(l.By, e):
		for _, md := range markdown.Render(l.Text, text, false) {
			out = append(out, strings.TrimPrefix(md, markdown.Indent))
		}
	default:
		for _, raw := range plainLines(l.Text) {
			out = append(out, cells.Lines(markdown.Inline(raw), text)...)
		}
	}

	return out
}

// supervisorThinking is the engine's turn before it has said anything.
func (s State) supervisorThinking(cw int, e Env) []string {
	eng := e.Engine

	var (
		dim  = theme.Paint(theme.Dim)
		rail = theme.Paint(theme.Live).Render("▎")
		when = dim.Render(e.Now.Format("15:04:05"))
		who  = theme.Paint(theme.Live).Bold(true).Render(eng)
	)

	head := railed(rail, when+"  "+who+" "+dim.Render("[supervisor]"))
	body := railed(rail, e.Spinner(theme.Live)+
		dim.Render(e.Words.T("supervisor.thinking", "supervisor is thinking...")))

	return []string{cells.Fit(head, cw), cells.Fit(body, cw), ""}
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
func isEngineName(by string, e Env) bool {
	if by == "supervisor" || (e.IsEngine != nil && e.IsEngine(by)) {
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
