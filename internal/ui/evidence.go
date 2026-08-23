package ui

// The evidence tab: what the record actually collected, per phase, per
// attempt.
//
// The log says a phase failed. This says which engine and model ran it, what
// it cost, which session it can be resumed from, why it stopped, and what it
// printed — quoted whole, never summarised. That last rule is the reason the
// tab exists: a pane that summarises an engine's output is a pane a reader
// cannot use to decide whether the engine was right, and deciding that is
// the entire job this window exists to support.
//
// Where the output was cut, the pane says so with both numbers. It does not
// go looking for the marker at the end of the text: internal/task writes the
// full size into the record when, and only when, it cut something, and
// view.Entry carries both sizes — so the question "was anything lost" is
// answered by arithmetic on two integers rather than by matching a sentence
// that could be rephrased.

import (
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// quoteMark is what the engine's own words are set behind, so that a line of
// output that happens to begin with a dash is not read as part of the
// window's own furniture.
const quoteMark = "│ "

// evidenceLines is the evidence tab's content, ready for the pane.
func (m Model) evidenceLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{" " + Paint(Bad).Render(m.logErr.Error())}
	}
	w, blocks := max(m.frame.Body.W, 1), 0
	var out []string
	var started view.Entry
	for _, e := range m.entries {
		if e.Attempted() {
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
		out = append(out, m.phaseHead(e, started))
		out = append(out, m.phaseBody(e)...)
	}
	if blocks == 0 {
		return []string{" " + Paint(Dim).Render(p.T("evidence.empty", "no phase of this task has run yet"))}
	}
	return out
}

// phaseHead is one phase's standing facts on one line: which phase, what ran
// it, what it cost and which session it left behind.
//
// The engine and the model come from the phase.started event and not from
// the one that ended it, because that is where internal/task writes them. A
// phase whose start is missing from the record — a log truncated at the
// front, a build older than the field — shows the phase without them rather
// than guessing from the task's own.
func (m Model) phaseHead(e, started view.Entry) string {
	p := m.opts.Words
	parts := []string{Paint(Accent).Render(e.Phase)}
	if engine := strings.TrimSpace(started.Engine + " " + started.Model); engine != "" {
		parts = append(parts, Paint(Dim).Render(engine))
	}
	if e.Cost > 0 {
		// strconv.FormatFloat always writes a full stop; Spanish writes a
		// comma here instead. group, below in this file, makes the
		// identical simplification for thousands separators and defends it
		// at length — this is the same trade over a different number: two
		// fixed decimal places, read once, never summed against anything
		// else on the row, so the wrong separator costs a moment's
		// unfamiliarity rather than the table of rules a per-language
		// formatter would need for one figure on one screen.
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
		// Both numbers, because one of them is the question. The rest of
		// the output was never written anywhere: plan 1 named phases/<n>/
		// as the place it would live and did not build it, and a pane that
		// implied the full text was a keystroke away would be sending the
		// reader to look for a file that does not exist.
		out = append(out, "    "+Paint(Warn).Render(p.T("evidence.truncated",
			"{kept} of {full} bytes kept — the rest was not written down anywhere",
			about("kept", group(e.Kept)), about("full", group(e.Full)))))
	}
	if strings.TrimSpace(e.Text) == "" {
		return append(out, "    "+Paint(Dim).Render(p.T("evidence.silent", "the engine printed nothing")))
	}
	for _, line := range strings.Split(strings.TrimSuffix(e.Text, "\n"), "\n") {
		out = append(out, "    "+Paint(Dim).Render(quoteMark)+line)
	}
	return out
}

// group puts a separator every three digits, so that a number a reader is
// meant to compare against another number can be compared at a glance.
//
// The separator is a comma in every language this program ships, which is a
// simplification and is written down as one: Spanish groups with a full stop
// and separates decimals with a comma. These are byte counts — whole
// numbers, no decimal part, never summed by the reader — so the cost of the
// wrong separator is a moment's unfamiliarity, where the cost of a
// per-language number formatter is a table of rules in a package that draws
// rows.
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
