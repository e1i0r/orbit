package panes

import (
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/ui/markdown"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// Report is what the engine wrote about the change and what the review
// concluded, and beside it which attempt each seam belongs to — laid out in
// one pass for the reason the timeline is.
func Report(e Env) ([]string, map[int]int) {
	p := e.Words
	if e.Failed != "" {
		return []string{"  " + theme.Paint(theme.Bad).Render(e.Failed)}, nil
	}

	w, blocks := max(e.Frame.Body.W, 1), 0

	var out []string

	out = append(out,
		"",
		"  "+theme.Paint(theme.Accent).Bold(true).Render(p.T("report.title", "Summary Report & Review")),
		"  "+theme.Paint(theme.Dim).Render(p.T("report.subtitle", "what it wrote about the change, and what the review concluded")),
		"",
	)

	var started view.Entry

	seams := map[int]int{}

	for _, entry := range e.Entries {
		if entry.Attempted() {
			seams[len(out)] = entry.Attempt
			out = append(out, e.seam(entry, w))
			started = view.Entry{}

			continue
		}

		if entry.Phase == "" {
			continue
		}

		if entry.What() == view.EntryStarted {
			started = entry
			continue
		}

		switch entry.What() {
		case view.EntryFinished, view.EntryFailed, view.EntryCancelled:
		default:
			continue
		}

		blocks++

		// Counted before it is drawn, so that the pane says it holds no
		// report when the record holds none and never because the reader
		// shut the attempts it does hold.
		if !e.attempt(entry.Attempt) {
			continue
		}

		out = append(out, e.phaseHead(entry, started))
		out = append(out, e.phaseBody(entry)...)
	}

	if blocks == 0 {
		return []string{"  " + theme.Paint(theme.Dim).Render(p.T("report.empty", "no engine report available for this task"))}, nil
	}

	return out, seams
}

// phaseHead is one phase's standing facts on one line.
func (e Env) phaseHead(entry, started view.Entry) string {
	p := e.Words

	parts := []string{theme.Paint(theme.Accent).Render(entry.Phase)}
	if engine := strings.TrimSpace(started.Engine + " " + started.Model); engine != "" {
		parts = append(parts, theme.Paint(theme.Dim).Render(engine))
	}

	if entry.Cost > 0 {
		parts = append(parts, theme.Paint(theme.Dim).Render(p.T("evidence.cost", "cost ${amount}",
			about("amount", strconv.FormatFloat(entry.Cost, 'f', 2, 64)))))
	}

	if entry.Session != "" {
		parts = append(parts, theme.Paint(theme.Dim).Render(p.T("evidence.session", "session {id}",
			about("id", entry.Session))))
	}

	word, role := e.logWord(entry)

	return "  " + theme.Paint(role).Render(word) + "  " + strings.Join(parts, "  ")
}

// phaseBody is why the phase stopped and what it printed.
func (e Env) phaseBody(entry view.Entry) []string {
	p := e.Words

	var out []string
	if entry.Cause != "" {
		out = append(out, "    "+theme.Paint(theme.Bad).Render(
			p.T("evidence.stopped", "stopped: {why}", about("why", entry.Cause))))
	}

	if entry.Truncated() {
		out = append(out, "    "+theme.Paint(theme.Warn).Render(p.T("evidence.truncated",
			"{kept} of {full} bytes kept — the rest was not written down anywhere",
			about("kept", group(entry.Kept)), about("full", group(entry.Full)))))
	}

	// Raw is the record itself, framing and all, because a reader who asks
	// for raw is asking what was written down and not what was made of it.
	text := entry.Said()
	if e.Raw {
		text = entry.Text
	}

	if strings.TrimSpace(text) == "" {
		return append(out, "    "+theme.Paint(theme.Dim).Render(p.T("evidence.silent", "the engine printed nothing")))
	}

	return append(out, markdown.Render(text, e.Frame.Body.W, e.Raw)...)
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
