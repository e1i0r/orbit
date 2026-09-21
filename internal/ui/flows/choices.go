package flows

// What a dial offers, in one place.
//
// The designer draws its short dials as a row of pills — the template, the
// engine, how hard the phase thinks, what it is handed and what happens
// when it ends — and a click on one of them chose nothing: the hit-test
// answered the row, and the row's action steps the dial one along. So
// clicking "off" on a dial sitting on "adaptive" moved it to "on", and the
// reader clicked again, and watched.
//
// The options are read from here by both the drawing and the pointer, which
// is what lets a click land on the pill it is over: two lists written twice
// would be two readings of one dial, and the click would choose whatever
// the older of them had in that column.

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// choice is one option of a dial: the id it sets, and the word it is drawn
// as when those are not the same string.
type choice struct {
	id    string
	label string
}

// valueAt is the column a field's value starts in: the mark the cursor
// stands in, the label column, and the space after it.
const valueAt = 2 + labelWidth + 1

// templateNames is the presets the designer offers. It is a list of its own
// because applyFlowTemplate is what knows the phases behind each name.
func templateNames() []string {
	return []string{"ninguna", "TDD Fuzz & PR", "TDD Cycle", "Security Audit", "Turbo Fix"}
}

// choices is what one dial offers and which option is on it now.
//
// Only the dials drawn as pills: the model and the effort can run to sixty
// options, which is a list and not a row, and a click on one of those rows
// opens that list.
func (s State) choices(field int, e Env) (opts []choice, current string, ok bool) {
	p := e.Words

	switch field {
	case flowFieldTemplate:
		return plainChoices(templateNames()), s.template, true
	case flowFieldEngine:
		return plainChoices(e.Engines()), cells.OrDef(s.edited().Engine, e.Engine), true
	case flowFieldThinking:
		return plainChoices([]string{"adaptive", "on", "off"}),
			cells.OrDef(s.edited().Thinking, "adaptive"), true
	case flowFieldFeedOutput:
		off, on := p.T("flows.feed_off", "starts fresh"), p.T("flows.feed_on", "takes the last output")

		return plainChoices([]string{off, on}), pickOne(s.edited().FeedOutput, off, on), true
	case flowFieldWait:
		auto, human := p.T("flows.wait_auto", "carries on"), p.T("flows.wait_human", "wait (human)")

		return plainChoices([]string{auto, human}), pickOne(s.cur().Wait, auto, human), true
	case flowFieldIsLoop:
		no, yes := p.T("flows.repeat_no", "runs once"), p.T("flows.repeat_yes", "repeats ↻")

		return plainChoices([]string{no, yes}), pickOne(s.looping(), no, yes), true
	}

	return nil, "", false
}

// pickOne is the word for a dial of two, which is what a flag is drawn as.
func pickOne(on bool, off, yes string) string {
	if on {
		return yes
	}

	return off
}

// plainChoices is a dial whose ids are the words it is drawn as.
func plainChoices(ids []string) []choice {
	out := make([]choice, 0, len(ids))
	for _, id := range ids {
		out = append(out, choice{id: id, label: id})
	}

	return out
}

// renderChoices is the row of pills, the one in use filled.
func renderChoices(opts []choice, current string) string {
	views := make([]string, 0, len(opts))

	for _, o := range opts {
		if o.id == current {
			views = append(views, theme.Paint(theme.Sel).Render(" "+o.label+" "))

			continue
		}

		views = append(views, theme.Paint(theme.Dim).Render(o.label))
	}

	return strings.Join(views, " ")
}

// choiceAt is which pill of a dial's row a column is over, and whether it is
// over one at all.
//
// It walks the same pills renderChoices draws, in the same order and at the
// same widths, from the column a field's value starts in. The gap between
// two of them belongs to neither: a click there is a click on the row.
func choiceAt(x int, opts []choice, current string) (int, bool) {
	at := valueAt

	for i, o := range opts {
		wide := lipgloss.Width(o.label)
		if o.id == current {
			wide += 2 // the space either side of the one in use
		}

		if x >= at && x < at+wide {
			return i, true
		}

		at += wide + 1 // the space renderChoices joins them with
	}

	return 0, false
}

// phaseLabels is what the "Editing phase" dial draws: one per phase, the
// number and the name, and the mark a loop carries.
func (s State) phaseLabels() []string {
	out := make([]string, 0, len(s.phases))

	for i, ph := range s.phases {
		label := strconv.Itoa(i+1) + "." + ph.Name
		if ph.Loop != nil {
			label += " ↻"
		}

		out = append(out, label)
	}

	return out
}

// renderPhaseTabs is that row drawn, the phase being edited marked.
func (s State) renderPhaseTabs() string {
	labels := s.phaseLabels()
	tabs := make([]string, 0, len(labels))

	for i, label := range labels {
		if i == s.activePhase {
			tabs = append(tabs, theme.Paint(theme.Sel).Bold(true).Render(" ● "+label+" "))

			continue
		}

		tabs = append(tabs, theme.Paint(theme.Dim).Render(" "+label+" "))
	}

	return strings.Join(tabs, " ")
}

// phaseTabAt is which phase of that row a column is over.
//
// The same walk as choiceAt, over the same widths: the one being edited
// carries a mark and is two cells wider for it.
func (s State) phaseTabAt(x int) (int, bool) {
	at := valueAt

	for i, label := range s.phaseLabels() {
		wide := lipgloss.Width(label) + 2
		if i == s.activePhase {
			wide += 2 // the mark and the space after it
		}

		if x >= at && x < at+wide {
			return i, true
		}

		at += wide + 1
	}

	return 0, false
}

// setChoice puts one dial on the option a click landed on.
//
// The field is chosen as well as the value, because a reader who pointed at
// a dial is working on that dial: leaving the cursor where it was would
// send the next arrow key somewhere else on the form.
func (s State) setChoice(field, at int, e Env) (State, Out) {
	s.field = field
	s.ensurePhase()

	opts, current, ok := s.choices(field, e)
	if !ok || at < 0 || at >= len(opts) {
		return s, Out{}
	}

	want := opts[at]
	if want.id == current {
		return s, Out{}
	}

	switch field {
	case flowFieldTemplate:
		s.template = want.id

		return s.applyFlowTemplate(s.template, e)
	case flowFieldEngine:
		s.edited().Engine = want.id
	case flowFieldThinking:
		s.edited().Thinking = want.id
	case flowFieldFeedOutput:
		s.edited().FeedOutput = at == 1
	case flowFieldWait:
		s.cur().Wait = at == 1
	case flowFieldIsLoop:
		// A loop is a phase of another shape rather than a flag, so the
		// dial is turned by the one function that knows how to put a
		// phase into one and take it back out.
		return s.toggleLoop(), Out{}
	}

	return s, Out{}
}
