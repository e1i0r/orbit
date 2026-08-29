package ui

import (
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// quoteMark is what the engine's own words are set behind.
const quoteMark = "│ "

// reportLines renders Pane 7: The engine's summary reports and conclusions.
func (m Model) reportLines() []string {
	lines, _ := m.reportRows()

	return lines
}

// reportRows is that content and, beside it, which attempt each seam belongs
// to, laid out in one pass for the reason logRows is.
func (m Model) reportRows() ([]string, map[int]int) {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.errSaid(m.logErr))}, nil
	}

	w, blocks := max(m.frame.Body.W, 1), 0

	var out []string

	out = append(out,
		"",
		"  "+Paint(Accent).Bold(true).Render(p.T("report.title", "Summary Report & Review")),
		"  "+Paint(Dim).Render(p.T("report.subtitle", "what it wrote about the change, and what the review concluded")),
		"",
	)

	var started view.Entry

	seams := map[int]int{}

	for _, e := range m.entries {
		if e.Attempted() {
			seams[len(out)] = e.Attempt
			out = append(out, m.seam(e, w))
			started = view.Entry{}

			continue
		}

		if e.Phase == "" {
			continue
		}

		if e.What() == view.EntryStarted {
			started = e
			continue
		}

		switch e.What() {
		case view.EntryFinished, view.EntryFailed, view.EntryCancelled:
		default:
			continue
		}

		blocks++

		// Counted before it is drawn, so that the pane says it holds no
		// report when the record holds none and never because the reader
		// shut the attempts it does hold.
		if !m.attemptOpen(e.Attempt) {
			continue
		}

		out = append(out, m.phaseHead(e, started))
		out = append(out, m.phaseBody(e)...)
	}

	if blocks == 0 {
		return []string{"  " + Paint(Dim).Render(p.T("report.empty", "no engine report available for this task"))}, nil
	}

	return out, seams
}

// phaseHead is one phase's standing facts on one line.
func (m Model) phaseHead(e, started view.Entry) string {
	p := m.opts.Words

	parts := []string{Paint(Accent).Render(e.Phase)}
	if engine := strings.TrimSpace(started.Engine + " " + started.Model); engine != "" {
		parts = append(parts, Paint(Dim).Render(engine))
	}

	if e.Cost > 0 {
		parts = append(parts, Paint(Dim).Render(p.T("evidence.cost", "cost ${amount}",
			about("amount", strconv.FormatFloat(e.Cost, 'f', 2, 64)))))
	}

	if e.Session != "" {
		parts = append(parts, Paint(Dim).Render(p.T("evidence.session", "session {id}",
			about("id", e.Session))))
	}

	word, role := m.logWord(e)

	return "  " + Paint(role).Render(word) + "  " + strings.Join(parts, "  ")
}

// phaseBody is why the phase stopped and what it printed.
func (m Model) phaseBody(e view.Entry) []string {
	p := m.opts.Words

	var out []string
	if e.Cause != "" {
		out = append(out, "    "+Paint(Bad).Render(p.T("evidence.stopped", "stopped: {why}", about("why", e.Cause))))
	}

	if e.Truncated() {
		out = append(out, "    "+Paint(Warn).Render(p.T("evidence.truncated",
			"{kept} of {full} bytes kept — the rest was not written down anywhere",
			about("kept", group(e.Kept)), about("full", group(e.Full)))))
	}

	// Raw is the record itself, framing and all, because a reader who asks
	// for raw is asking what was written down and not what was made of it.
	text := e.Said()
	if m.rawText {
		text = e.Text
	}

	if strings.TrimSpace(text) == "" {
		return append(out, "    "+Paint(Dim).Render(p.T("evidence.silent", "the engine printed nothing")))
	}

	return append(out, renderMarkdown(text, m.frame.Body.W, m.rawText)...)
}

func group(n int) string {
	digits := []rune(strconv.Itoa(n))

	var b strings.Builder

	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}

		b.WriteRune(r)
	}

	return b.String()
}
