package known

// The list as a table: one rule to a row, and what a reader wants off it
// lined up under headings.
//
// Two columns where there was one. What a rule does — say its sentence, or
// refuse the work — and where it stands — applying, stopped for now, decided
// against — are different questions, and a paused rule is still the one that
// will refuse the work when it comes back. A single field that said either
// "paused" or "stops" could only answer one of them, so a reader looking
// down it for what is actually stopping their runs was reading a column that
// hid half of them.
//
// Columns and not a sentence with dots in it, because this list is scanned
// rather than read: the question it is opened with is "which of these is in
// my way", and that is answered by an eye going straight down one column.

import (
	"strconv"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/fact"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// The columns, in cells, and the space between two of them. The sentence is
// not among them: it is prose, and it takes whatever the others left.
const (
	colName  = 8
	colDoes  = 9
	colState = 7
	colWhere = 16
	colGap   = 2
	// saysLeast is the narrowest the sentence is allowed to get before the
	// place starts giving up cells to it. A rule read four words to a line
	// is a rule nobody reads.
	saysLeast = 28
	// nameFrom is the width below which the name is not drawn. An id is
	// what a rule is called on the command line and it earns ten cells of a
	// wide terminal; on a narrow one the sentence needs them more.
	nameFrom = 78
	// gutter is the two cells every row opens with: the cursor's mark, or
	// the dot that says this one is waiting on somebody.
	gutter = 2
)

// paint is a colour, as the one thing a column does with it.
type paint func(...string) string

// widths is how many cells each column got at the width the screen has.
type widths struct{ name, does, state, where, says int }

// plan shares the width out. Every column but the sentence holds a word or a
// path and has a size it wants; the sentence gets the rest, and takes cells
// off the place rather than go under saysLeast.
func plan(cw int) widths {
	w := widths{does: colDoes, state: colState}
	if cw >= nameFrom {
		w.name = colName
	}

	rest := cw - gutter - w.does - colGap - w.state - colGap
	if w.name > 0 {
		rest -= w.name + colGap
	}

	w.where = max(6, min(colWhere, rest-colGap-saysLeast))
	w.says = max(12, rest-colGap-w.where)

	return w
}

// left is the cells before the sentence, which is where a line the sentence
// wrapped onto has to start.
func (w widths) left() int {
	n := gutter + w.does + colGap + w.state + colGap + w.where + colGap
	if w.name > 0 {
		n += w.name + colGap
	}

	return n
}

// heading is the row of column names.
//
// Once, above the first group, and not repeated over each one. The words are
// furniture: they say what the columns are to somebody meeting the screen,
// and after that they are four cells of grey the eye skips.
func heading(w widths, e Env) string {
	p := e.Words
	dim := theme.Paint(theme.Dim).Render

	return strings.Repeat(" ", gutter) +
		col(p.T("knowledge.col_name", "NAME"), w.name, dim) +
		col(p.T("knowledge.col_does", "DOES"), w.does, dim) +
		col(p.T("knowledge.col_state", "STATE"), w.state, dim) +
		col(p.T("knowledge.col_where", "WHERE"), w.where, dim) +
		dim(p.T("knowledge.col_rule", "THE RULE"))
}

// col is one cell of a row: cut to its budget, filled out to it, and
// coloured. A column planned at nothing is not drawn at all.
func col(text string, n int, ink paint) string {
	if n <= 0 {
		return ""
	}

	return ink(cells.Pad(text, n, false)) + strings.Repeat(" ", colGap)
}

// factRow is one rule: its columns, the sentence wrapped under the last of
// them, and — only while the cursor is on it — the dim lines that say where
// it came from and what it was paused for.
//
// The provenance is shown for one row rather than for all of them because
// both are true at once: a sentence nobody can trace is worth nothing, and a
// column of dates beside every rule is a column nobody reads. The row being
// looked at is the row the question is about.
func (s State) factRow(f knowledge.Fact, chosen bool, w widths, e Env) []string {
	ink := theme.Text(theme.Primary).Render
	if !f.Tells() {
		ink = theme.Paint(theme.Dim).Render
	}

	doesText, doesInk := does(f, e)
	stateText, stateInk := stands(f, e)

	said := cells.Lines(f.Phrase, w.says)
	if len(said) == 0 {
		said = []string{""}
	}

	rows := []string{
		rowMark(f, chosen) +
			col(f.ID, w.name, theme.Paint(theme.Dim).Render) +
			col(doesText, w.does, doesInk) +
			col(stateText, w.state, stateInk) +
			col(cells.Tail(fact.Where(f.Scope), w.where), w.where, theme.Text(theme.Secondary).Render) +
			ink(said[0]),
	}

	pad := strings.Repeat(" ", w.left())
	for _, line := range said[1:] {
		rows = append(rows, pad+ink(line))
	}

	if !chosen {
		return rows
	}

	for _, line := range s.aside(f, w.says, e) {
		rows = append(rows, pad+theme.Paint(theme.Dim).Render(line))
	}

	return rows
}

// rowMark is the two cells a row opens with: the cursor, the dot that says
// this rule is waiting to be decided about, or nothing.
//
// A dot and not a word. What is being answered is "is there anything here
// for me", and a mark down the edge answers it at a glance from across the
// room, where a fourth column of words would have to be read.
func rowMark(f knowledge.Fact, chosen bool) string {
	switch {
	case chosen:
		return theme.Paint(theme.Accent).Bold(true).Render(cells.Mark + " ")
	case f.Review:
		return theme.Paint(theme.Warn).Render("• ")
	}

	return strings.Repeat(" ", gutter)
}

// aside is what the row under the cursor says beyond its columns: where the
// rule came from, and what it was paused for.
func (s State) aside(f knowledge.Fact, cw int, e Env) []string {
	out := cells.Lines(from(f, e), cw)

	// The one thing on this screen worth spelling out where it is. A rule
	// that asked to stop and brought no command reads like a gate in every
	// list it appears in and is not one, and the reader who needs to know
	// that is the reader with the cursor on it.
	if f.Stops && f.Check == "" {
		out = append(out, cells.Lines(e.Words.T("knowledge.wants_a_check",
			"it asked to stop the work and has no command to stop it with, "+
				"so it only says its sentence"), cw)...)
	}

	if f.Why != "" {
		out = append(out, cells.Lines(e.Words.T("knowledge.paused_for",
			"paused: {why}", about("why", f.Why)), cw)...)
	}

	return out
}

// does is what the rule does when the work reaches its scope: refuse it, or
// put its sentence in front of the agent.
//
// A rule that asked to stop and brought no check is neither, and says so. It
// would never fire while reading as though it would, which is the one thing
// on this screen worth being loud about.
func does(f knowledge.Fact, e Env) (string, paint) {
	p := e.Words

	switch {
	case f.Action() == knowledge.Stops:
		return p.T("knowledge.stops", "stops"), theme.Paint(theme.Bad).Bold(true).Render
	case f.Stops:
		return p.T("knowledge.no_check", "no check"), theme.Paint(theme.Warn).Render
	default:
		return p.T("knowledge.says", "says"), theme.Paint(theme.Live).Render
	}
}

// stands is where the rule stands with the reader: applying, stopped for
// now, or decided against.
//
// Applying is drawn as furniture rather than in a colour. It is the ordinary
// case and there is nothing to decide about it, and a column where every row
// is lit is a column that has stopped pointing at anything.
func stands(f knowledge.Fact, e Env) (string, paint) {
	p := e.Words

	switch f.State {
	case knowledge.Paused:
		return p.T("knowledge.is_paused", "paused"), theme.Paint(theme.Warn).Render
	case knowledge.Off:
		return p.T("knowledge.is_off", "off"), theme.Paint(theme.Dim).Render
	default:
		return p.T("knowledge.is_active", "active"), theme.Text(theme.Tertiary).Render
	}
}

// from is where a fact came from, and how much use it has had.
//
// The source is the whole of why a fact can be trusted: one read off the code
// regenerates itself, one somebody said is somebody's opinion with a name on
// it, and one the record produced came out of a refusal that actually
// happened. A sentence nobody can trace is one the model may as well have
// made up.
func from(f knowledge.Fact, e Env) string {
	p := e.Words

	said := map[knowledge.Source]string{
		knowledge.FromCode:       p.T("knowledge.from_code", "from the code"),
		knowledge.Human:          p.T("knowledge.from_human", "you said it"),
		knowledge.FromRecord:     p.T("knowledge.from_record", "from the record"),
		knowledge.FromProduction: p.T("knowledge.from_prod", "from production"),
		knowledge.FromDocs:       p.T("knowledge.from_docs", "the project already said it"),
		knowledge.FromHistory:    p.T("knowledge.from_history", "the history says so"),
	}[f.Source]

	if f.Ref != "" {
		said += cells.Dot + f.Ref
	}

	// The date and not "three days ago": a fact is not read in a hurry, and
	// a date can be looked up against a pull request or a task while a
	// relative age cannot.
	if !f.At.IsZero() {
		said += cells.Dot + f.At.Format(time.DateOnly)
	}

	if f.Used > 0 {
		said += cells.Dot + p.T("knowledge.used", "told {n}×", about("n", strconv.Itoa(f.Used)))
	}

	return said
}

// factRepo is the checkout a rule belongs to, by the name the screen calls
// it: the last segment of the path, which is what anybody would say out loud.
func factRepo(f knowledge.Fact) string { return fact.Repo(f.Scope.Repo) }
