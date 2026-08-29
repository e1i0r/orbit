package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

type thoughtBlock struct {
	at      string
	phase   string
	attempt int
	lines   []string

	// entry is where in the record the block was written, which is what the
	// pane opens and closes it by. The block for the thought in flight has
	// not been written down yet, and is -1.
	entry int
}

// formatThoughtLine formats a thought line into a clean, concise decision bullet.
func formatThoughtLine(l string) (string, Role) {
	lower := strings.ToLower(l)
	switch {
	case strings.Contains(lower, "decid") || strings.Contains(lower, "conclu") ||
		strings.Contains(lower, "opt") || strings.Contains(lower, "resolv") ||
		strings.Contains(lower, "implement"):
		return "🎯 " + l, OK
	case strings.Contains(lower, "descart") || strings.Contains(lower, "turn down") ||
		strings.Contains(lower, "reject") || strings.Contains(lower, "avoid") ||
		strings.Contains(lower, "evit"):
		return "🚫 " + l, Warn
	case strings.Contains(lower, "investig") || strings.Contains(lower, "check") ||
		strings.Contains(lower, "find") || strings.Contains(lower, "encontr") ||
		strings.Contains(lower, "analiz") || strings.Contains(lower, "evalu"):
		return "🔍 " + l, Live
	case strings.Contains(lower, "porqu") || strings.Contains(lower, "becaus") ||
		strings.Contains(lower, "razon") || strings.Contains(lower, "reason") ||
		strings.Contains(lower, "motivo"):
		return "💡 " + l, Accent
	default:
		return "• " + l, Dim
	}
}

// thinkingLines renders Pane 11: Concise decision reasoning and analysis captured from the model.
func (m Model) thinkingLines() []string {
	lines, _ := m.thinkingRows()

	return lines
}

// thinkingRows is that content and, beside it, which entry each block that
// folds was written by, laid out in one pass for the reason logRows is.
func (m Model) thinkingRows() ([]string, map[int]int) {
	p := m.opts.Words

	blocks := m.thoughtBlocks()

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("thinking.title", "Decision Analysis & Agent Thinking")),
		"  " + Paint(Dim).Render(p.T("thinking.subtitle", "why it made each decision, what it evaluated and what it turned down")),
		"",
	}

	if len(blocks) == 0 {
		return append(out, "  "+Paint(Dim).Render(p.T("thinking.empty",
			"no thinking blocks or decision logs captured for this task"))), nil
	}

	out = append(out, fmt.Sprintf("  %d %s",
		len(blocks),
		p.T("thinking.entries_count", "reasoning and decisions analysed"),
	), "")

	heads, w := map[int]int{}, max(m.frame.Body.W, 1)

	for _, b := range blocks {
		rows, folds := m.thoughtRows(b, w)
		if folds {
			heads[len(out)] = b.entry
		}

		out = append(out, rows...)
		out = append(out, "")
	}

	return out, heads
}

// thoughtBlocks is the record read as reasoning: the thinking the engine
// showed its work in, the summary each phase ended on, and the sentence the
// one that is running is on right now.
func (m Model) thoughtBlocks() []thoughtBlock {
	var blocks []thoughtBlock

	for i, e := range m.entries {
		isThought := e.What() == view.EntryThought

		// Said and not Text: a phase that was killed leaves its stream on
		// stdout, and the first line of a folded block would be a line of
		// somebody's JSON rather than the sentence it stands for.
		said := strings.TrimSpace(e.Said())

		isPhaseSummary := (e.What() == view.EntryFinished || e.What() == view.EntryFailed) && said != ""
		if !isThought && !isPhaseSummary {
			continue
		}

		timeStr := ""
		if !e.At.IsZero() {
			timeStr = e.At.Format("15:04:05")
		}

		var lines []string

		for _, l := range strings.Split(said, "\n") {
			if l = strings.TrimSpace(l); l != "" {
				lines = append(lines, l)
			}
		}

		if len(lines) > 0 {
			blocks = append(blocks, thoughtBlock{
				at: timeStr, phase: e.Phase, attempt: e.Attempt, lines: lines, entry: i,
			})
		}
	}

	t, ok := m.task(m.detail)
	if ok && t.CurrentThought != "" {
		blocks = append(blocks, thoughtBlock{
			at:    m.opts.Words.T("thinking.live_now", "live / in flight"),
			phase: t.Phase, lines: []string{t.CurrentThought}, entry: -1,
		})
	}

	return blocks
}

// thoughtIndent is how far the reasoning is set in from the head that dates
// it, and thoughtGutter is the two cells in front of that head the arrow
// stands in — spaces on a block that has nothing to open, so that every
// block on the tab begins in the same column.
const (
	thoughtIndent = "      "
	thoughtGutter = "  "
)

// thoughtRows is one block — when it was thought, in which phase and attempt,
// and what it says — and whether it has more to say than it is showing.
//
// Whether it folds is decided here, by wrapping the reasoning at the measure
// it will be drawn at: a block that says all it has in one row is offered no
// arrow, because opening it would put nothing new on the screen.
func (m Model) thoughtRows(b thoughtBlock, w int) ([]string, bool) {
	p := m.opts.Words

	head := Paint(Accent).Render("●") + " " + Paint(Dim).Render(b.at)
	if b.phase != "" {
		head += "  " + Paint(Accent).Render(b.phase)
	}

	if b.attempt > 0 {
		head += "  " + Paint(Dim).Render(p.P("thinking.attempt", b.attempt, "attempt {n}", "attempt {n}"))
	}

	availW := max(20, w-len(thoughtIndent)-2)

	var body []string

	for _, l := range b.lines {
		formatted, role := formatThoughtLine(l)
		for _, wl := range splitIntoLines(formatted, availW) {
			body = append(body, thoughtIndent+Paint(role).Render(fit(wl, availW)))
		}
	}

	if len(body) <= 1 {
		return append([]string{"  " + thoughtGutter + head}, body...), false
	}

	open := m.rowOpen(tabThinking, b.entry)
	out := []string{"  " + Text(Tertiary).Render(foldMark(open)) + head}

	if !open {
		return append(out, body[0]), true
	}

	return append(out, body...), true
}
