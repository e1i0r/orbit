package flows

// The designer's fields, in four groups.
//
// Grouped rather than numbered. The form was one run of eleven numbered
// rows, and the numbers were written into the labels by hand — so a field
// that appears only for a phase that repeats either broke the numbering or
// was left out of the form altogether. What the groups say instead is what
// each run of fields is about: the flow, this phase, who runs it, and how it
// is joined to the phase before.

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// labelWidth is how wide the left column is. Every label is padded to it so
// that the values line up in one column a reader's eye can run down.
const labelWidth = 24

// builderFieldRows is every field from the template down to the phase's
// control, the loop's own among them when the phase repeats.
func (s State) builderFieldRows(w int, sz boxSizes, e Env) []builderLine {
	p := e.Words

	out := []builderLine{s.groupRow(p.T("flows.group_flow", "THE FLOW · what it is called and what it is for"), w)}

	tpls := []string{"ninguna", "TDD Fuzz & PR", "TDD Cycle", "Security Audit", "Turbo Fix"}
	out = append(out,
		s.labelled(flowFieldTemplate, p.T("flows.field_template", "Template / Preset"), renderComboPills(tpls, s.template), w),
		s.labelled(flowFieldName, p.T("flows.field_flow_name", "Flow name"), s.typedValue(flowFieldName, s.flowName, e), w),
		s.labelled(flowFieldDescription, p.T("flows.field_description", "Purpose"),
			theme.Paint(theme.Dim).Render(p.T("flows.desc_hint", "one line, or several — shift+↵ for a new one")), w),
	)

	out = append(out, s.textBox(flowFieldDescription, s.description,
		p.T("flows.desc_placeholder", "(what is this flow for?)"), sz.desc, w)...)

	out = append(out,
		s.groupRow(p.T("flows.group_phase", "THE PHASE · one step of the pipeline"), w),
	)

	out = append(out, s.phaseFieldRows(w, sz, e)...)
	out = append(out, s.groupRow(p.T("flows.group_engine", "WHO RUNS IT"), w))
	out = append(out, s.engineFieldRows(w, e)...)
	out = append(out, s.groupRow(p.T("flows.group_wiring", "HOW IT JOINS THE PHASE BEFORE"), w))

	return append(out, s.wiringFieldRows(w, e)...)
}

// phaseFieldRows is which phase is being edited, its name, and whether it
// repeats.
func (s State) phaseFieldRows(w int, sz boxSizes, e Env) []builderLine {
	p := e.Words

	var tabs []string

	for i, ph := range s.phases {
		label := strconv.Itoa(i+1) + "." + ph.Name
		if ph.Loop != nil {
			label += " ↻"
		}

		if i == s.activePhase {
			tabs = append(tabs, theme.Paint(theme.Sel).Bold(true).Render(" ● "+label+" "))
			continue
		}

		tabs = append(tabs, theme.Paint(theme.Dim).Render(" "+label+" "))
	}

	out := []builderLine{
		s.labelled(flowFieldPhaseSelect, p.T("flows.field_editing_phase", "Editing phase"), strings.Join(tabs, " "), w),
		s.labelled(flowFieldPhaseName, p.T("flows.field_phase_name", "Phase name"), s.typedValue(flowFieldPhaseName, s.cur().Name, e), w),
	}

	no, yes := p.T("flows.repeat_no", "runs once"), p.T("flows.repeat_yes", "repeats ↻")

	val := no
	if s.looping() {
		val = yes
	}

	out = append(out, s.labelled(flowFieldIsLoop, p.T("flows.field_is_loop", "Repeat until it passes"),
		renderComboPills([]string{no, yes}, val), w))

	if !s.looping() {
		return out
	}

	return append(out, s.loopFieldRows(w, sz, e)...)
}

// engineFieldRows is the three dials of the build — engine, model, effort —
// and the thinking mode. They are the phase's own, or the phase inside the
// loop's, because a loop runs nothing itself.
func (s State) engineFieldRows(w int, e Env) []builderLine {
	p := e.Words

	eng := cells.OrDef(s.edited().Engine, e.Engine)
	mdls, mdlLabels := e.Models(eng)
	effs, effLabels := e.Efforts(eng)

	return []builderLine{
		s.labelled(flowFieldEngine, p.T("flows.field_engine", "Engine"), renderComboPills(e.Engines(), eng), w),
		s.labelled(flowFieldModel, p.T("flows.field_model", "Model"),
			s.dialValue(flowFieldModel, mdls, mdlLabels, s.edited().Model, e), w),
		s.labelled(flowFieldEffort, p.T("flows.field_effort", "Effort"),
			s.dialValue(flowFieldEffort, effs, effLabels, s.edited().Effort, e), w),
		s.labelled(flowFieldThinking, p.T("flows.field_thinking", "Thinking"),
			renderComboPills([]string{"adaptive", "on", "off"}, cells.OrDef(s.edited().Thinking, "adaptive")), w),
	}
}

// wiringFieldRows is what the phase is handed and what happens when it ends.
func (s State) wiringFieldRows(w int, e Env) []builderLine {
	p := e.Words

	off, on := p.T("flows.feed_off", "starts fresh"), p.T("flows.feed_on", "takes the last output")

	feed := off
	if s.edited().FeedOutput {
		feed = on
	}

	auto, human := p.T("flows.wait_auto", "carries on"), p.T("flows.wait_human", "wait (human)")

	wait := auto
	if s.cur().Wait {
		wait = human
	}

	return []builderLine{
		s.labelled(flowFieldFeedOutput, p.T("flows.field_feed_output", "Previous output"), renderComboPills([]string{off, on}, feed), w),
		s.labelled(flowFieldWait, p.T("flows.field_wait", "When it ends"), renderComboPills([]string{auto, human}, wait), w),
	}
}

// groupRow is one heading over a run of fields, with a rule after it so the
// eye finds the next group without reading the words again.
func (s State) groupRow(head string, w int) builderLine {
	line := "  " + theme.Paint(theme.Live).Bold(true).Render(head) + " "

	// Every rule ends in the same column, so the groups read as one
	// column of the form rather than as four ragged ones.
	if rule := min(w, 104) - lipgloss.Width(line) - 2; rule > 0 {
		line += theme.Paint(theme.Dim).Render(strings.Repeat("─", rule))
	}

	return plainLine(cells.Fit(line, w))
}

// labelled is one label and its value, with the cursor's mark when the
// reader is on it.
func (s State) labelled(field int, label, val string, w int) builderLine {
	mark, lbl := "  ", cells.Pad(label, labelWidth, false)

	if s.field == field {
		mark = theme.Paint(theme.Accent).Bold(true).Render("▸ ")
		lbl = theme.Paint(theme.Accent).Bold(true).Render(lbl)
	} else {
		lbl = theme.Paint(theme.Dim).Render(lbl)
	}

	return builderLine{text: cells.Fit(mark+lbl+" "+val, w), field: field, phase: noPhase, pick: noPick}
}

// typedValue is a one-line text field: what it holds, with the caret after
// it while it is the field being typed into.
func (s State) typedValue(field int, val string, e Env) string {
	if s.field == field {
		return theme.Paint(theme.Accent).Render(val + "█")
	}

	if val == "" {
		return theme.Paint(theme.Dim).Render(e.Words.T("flows.empty_field", "(empty)"))
	}

	return theme.Text(theme.Primary).Render(val)
}
