package ui

// The designer's rows, and what each of them is.
//
// One list, built once. The screen used to draw the form from one function
// and work out where the reader had clicked from a second — a table of line
// numbers, walked by hand, in flowsmouse.go. The two agreed for as long as
// nobody added a field: the loop's fields appear only for a phase that
// repeats, and a hand-kept table cannot know that. Now every row says what
// it is, the draw takes the text and the mouse takes the field, and a row
// that moves moves in both.

import (
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
)

// noField and noPhase are what a row that is a heading, a rule or a blank
// says about itself.
const (
	noField = -1
	noPhase = -1
	noPick  = -1
)

// builderLine is one drawn row of the designer, and what a click on it means.
type builderLine struct {
	text  string
	field int // the field of the form it belongs to, or noField
	phase int // the pipeline phase it points at, or noPhase
	// pick is the choice a picker row stands for, or noPick.
	pick int
	// head is the label row of a field drawn over several rows, as opposed
	// to the box under it. It is what tells a click on the instructions
	// label from a click inside the instructions: the pills that paste,
	// write and clear a prompt sit on the label row alone.
	head bool
}

// plainLine is a row nobody can click: a heading, a blank, a hint.
func plainLine(text string) builderLine {
	return builderLine{text: text, field: noField, phase: noPhase, pick: noPick}
}

// builderView is the form, and the row the window starts at.
//
// The form is taller than a short terminal, and fill cuts what does not fit
// — which on this screen would be the Save button. So the window follows the
// field the reader is on: everything above it scrolls away rather than
// everything below it being lost.
func (m Model) builderView(h, w int) (lines []builderLine, start int) {
	if m.flows.picker.open {
		return m.pickerLines(h, w), 0
	}

	lines = m.builderLines(w, boxSizes{prompt: promptRows, checks: checkRows})
	if h <= 0 || len(lines) <= h {
		return lines, 0
	}

	// The boxes give their rows up before the form starts scrolling: a
	// reader who has to scroll to reach Save is a reader who has not seen
	// it, and eight rows of instructions are worth less than the button
	// that writes them down.
	lines = m.builderLines(w, shrunk(len(lines)-h, m.flows.looping()))
	if len(lines) <= h {
		return lines, 0
	}

	// The last row of the field and not the first: a box is several rows,
	// and a window placed on its label cuts the end of what is being typed.
	at := 0

	for i, l := range lines {
		if l.field == m.flows.field {
			at = i
		}
	}

	// Two rows of room under the cursor, so the field being typed into is
	// never the last thing on the screen.
	if at >= h-2 {
		start = min(at-(h-3), len(lines)-h)
	}

	return lines, start
}

// boxSizes is how many rows the two paragraph fields are given.
type boxSizes struct {
	prompt int
	checks int
}

// shrunk is those two sizes with the overflow taken out of them, the
// instructions first because they are the taller of the two. Neither goes
// below two rows: a box of one row is a line, and this screen has those.
func shrunk(over int, looping bool) boxSizes {
	sz := boxSizes{prompt: max(promptRows-over, 2), checks: checkRows}

	if looping {
		sz.checks = max(checkRows-max(over-(promptRows-sz.prompt), 0), 2)
	}

	return sz
}

// builderLines is the whole form, top to bottom.
func (m Model) builderLines(w int, sz boxSizes) []builderLine {
	st := &m.flows
	st.ensurePhase()

	out := m.builderHead(w)
	out = append(out, m.builderPipeline(w)...)
	out = append(out, m.builderFieldRows(w, sz)...)
	out = append(out, m.builderPromptRows(w, sz)...)

	return append(out, m.builderActions(w)...)
}

// builderHead is the title and what this screen is for.
func (m Model) builderHead(w int) []builderLine {
	p := m.opts.Words

	title := p.T("flows.builder_create_title", "Flow Designer (Create New Flow)")

	switch {
	case m.flows.readOnly:
		title = p.T("flows.builder_preview_title", "Flow Inspector (Read-Only: {name})", about("name", m.flows.flowName))
	case m.flows.isEditing:
		title = p.T("flows.builder_edit_title", "Flow Designer (Edit: {name})", about("name", m.flows.flowName))
	}

	// One row and not four. The form is taller than a terminal, and every
	// row spent on saying what this screen is is a row the reader has to
	// scroll past to reach the button that saves what they wrote — what
	// each field does is said at the bottom, about the field they are on.
	return []builderLine{plainLine(fit("  "+Paint(Accent).Bold(true).Render(title)+"  "+
		Paint(Dim).Render(p.T("flows.builder_subtitle",
			"a flow is phases in order; each is an engine given instructions")), w))}
}

// builderPipeline is the flow as it will run, one row per phase, and every
// row of it selects the phase the fields below are about.
func (m Model) builderPipeline(w int) []builderLine {
	st := &m.flows
	p := m.opts.Words

	out := []builderLine{plainLine("  " + Paint(Live).Bold(true).Render(
		p.T("flows.builder_pipeline_title", "Pipeline (Click on a phase to edit it):")))}

	for i, ph := range st.phases {
		prefix := "    ➔ "
		if i == 0 {
			prefix = "    "
		}

		curMark := "○ "
		if i == st.activePhase {
			curMark = "● "
		}

		title := curMark + p.T("flows.phase_label", "Phase") + " " +
			strconv.Itoa(i+1) + ": " + ph.Name + " (" + phaseRuns(m, ph) + ")"

		if runsIn(ph).FeedOutput {
			title += " " + p.T("flows.feeds_input", "[feeds input]")
		}

		if ph.Loop != nil {
			title += " " + Paint(Warn).Render(p.T("flows.repeats_up_to",
				"↻ up to {n}×", about("n", strconv.Itoa(ph.Loop.Max))))
		}

		if ph.Wait {
			title += " " + p.T("flows.stops_human", "(stops for human)")
		}

		ink := Paint(Dim)
		if i == st.activePhase {
			ink = Paint(Accent).Bold(true)
		}

		out = append(out, builderLine{text: fit(prefix+ink.Render(title), w), field: noField, phase: i, pick: noPick})

		if prompt := runsIn(ph).Prompt; prompt != "" {
			out = append(out, builderLine{
				text:  "       " + Paint(Dim).Render(fit(`"`+flatten(prompt)+`"`, w-14)),
				field: noField,
				phase: i,
				pick:  noPick,
			})
		}
	}

	return out
}

// runsIn is the phase that actually runs: the one inside a loop, or the
// phase itself. The pipeline reads it so that a loop's row says which engine
// and which instructions it goes round with, rather than going blank the
// moment somebody turns the repeat switch on.
func runsIn(ph flow.Phase) flow.Phase {
	if ph.Loop != nil && len(ph.Loop.Phases) > 0 {
		return ph.Loop.Phases[0]
	}

	return ph
}

// phaseRuns is the engine and model a phase runs on, as the pipeline names
// them.
func phaseRuns(m Model, ph flow.Phase) string {
	inner := runsIn(ph)

	return orDef(inner.Engine, m.opts.Words.T("flows.no_engine", "no engine")) + "/" + orDef(inner.Model, "default")
}

// flatten is a paragraph as a single row of the pipeline: the newlines a
// multi-line prompt holds would otherwise break the row it is drawn on.
func flatten(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\n", " ")), " ")
}
