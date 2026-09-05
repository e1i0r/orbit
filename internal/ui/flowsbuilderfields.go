package ui

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
)

// labelWidth is how wide the left column is. Every label is padded to it so
// that the values line up in one column a reader's eye can run down.
const labelWidth = 24

// builderFieldRows is every field from the template down to the phase's
// control, the loop's own among them when the phase repeats.
func (m Model) builderFieldRows(w int, sz boxSizes) []builderLine {
	st := &m.flows
	p := m.opts.Words

	out := []builderLine{m.groupRow(p.T("flows.group_flow", "THE FLOW · what it is called and what it is for"), w)}

	tpls := []string{"ninguna", "TDD Fuzz & PR", "TDD Cycle", "Security Audit", "Turbo Fix"}
	out = append(out,
		m.labelled(flowFieldTemplate, p.T("flows.field_template", "Template / Preset"), renderComboPills(tpls, st.template), w),
		m.labelled(flowFieldName, p.T("flows.field_flow_name", "Flow name"), m.typedValue(flowFieldName, st.flowName), w),
		m.labelled(flowFieldDescription, p.T("flows.field_description", "Purpose"),
			Paint(Dim).Render(p.T("flows.desc_hint", "one line, or several — shift+↵ for a new one")), w),
	)

	out = append(out, m.textBox(flowFieldDescription, st.description,
		p.T("flows.desc_placeholder", "(what is this flow for?)"), sz.desc, w)...)

	out = append(out,
		m.groupRow(p.T("flows.group_phase", "THE PHASE · one step of the pipeline"), w),
	)

	out = append(out, m.phaseFieldRows(w, sz)...)
	out = append(out, m.groupRow(p.T("flows.group_engine", "WHO RUNS IT"), w))
	out = append(out, m.engineFieldRows(w)...)
	out = append(out, m.groupRow(p.T("flows.group_wiring", "HOW IT JOINS THE PHASE BEFORE"), w))

	return append(out, m.wiringFieldRows(w)...)
}

// phaseFieldRows is which phase is being edited, its name, and whether it
// repeats.
func (m Model) phaseFieldRows(w int, sz boxSizes) []builderLine {
	st := &m.flows
	p := m.opts.Words

	var tabs []string

	for i, ph := range st.phases {
		label := strconv.Itoa(i+1) + "." + ph.Name
		if ph.Loop != nil {
			label += " ↻"
		}

		if i == st.activePhase {
			tabs = append(tabs, Paint(Sel).Bold(true).Render(" ● "+label+" "))
			continue
		}

		tabs = append(tabs, Paint(Dim).Render(" "+label+" "))
	}

	out := []builderLine{
		m.labelled(flowFieldPhaseSelect, p.T("flows.field_editing_phase", "Editing phase"), strings.Join(tabs, " "), w),
		m.labelled(flowFieldPhaseName, p.T("flows.field_phase_name", "Phase name"), m.typedValue(flowFieldPhaseName, st.cur().Name), w),
	}

	no, yes := p.T("flows.repeat_no", "runs once"), p.T("flows.repeat_yes", "repeats ↻")

	val := no
	if st.looping() {
		val = yes
	}

	out = append(out, m.labelled(flowFieldIsLoop, p.T("flows.field_is_loop", "Repeat until it passes"),
		renderComboPills([]string{no, yes}, val), w))

	if !st.looping() {
		return out
	}

	return append(out, m.loopFieldRows(w, sz)...)
}

// engineFieldRows is the three dials of the build — engine, model, effort —
// and the thinking mode. They are the phase's own, or the phase inside the
// loop's, because a loop runs nothing itself.
func (m Model) engineFieldRows(w int) []builderLine {
	st := &m.flows
	p := m.opts.Words

	eng := m.dialEngine(st.edited().Engine)
	mdls, mdlLabels := m.modelsFor(eng)
	effs, effLabels := m.effortsFor(eng)

	return []builderLine{
		m.labelled(flowFieldEngine, p.T("flows.field_engine", "Engine"), renderComboPills(m.engineNames(), eng), w),
		m.labelled(flowFieldModel, p.T("flows.field_model", "Model"),
			m.dialValue(flowFieldModel, mdls, mdlLabels, st.edited().Model), w),
		m.labelled(flowFieldEffort, p.T("flows.field_effort", "Effort"),
			m.dialValue(flowFieldEffort, effs, effLabels, st.edited().Effort), w),
		m.labelled(flowFieldThinking, p.T("flows.field_thinking", "Thinking"),
			renderComboPills([]string{"adaptive", "on", "off"}, orDef(st.edited().Thinking, "adaptive")), w),
	}
}

// wiringFieldRows is what the phase is handed and what happens when it ends.
func (m Model) wiringFieldRows(w int) []builderLine {
	st := &m.flows
	p := m.opts.Words

	off, on := p.T("flows.feed_off", "starts fresh"), p.T("flows.feed_on", "takes the last output")

	feed := off
	if st.edited().FeedOutput {
		feed = on
	}

	auto, human := p.T("flows.wait_auto", "carries on"), p.T("flows.wait_human", "wait (human)")

	wait := auto
	if st.cur().Wait {
		wait = human
	}

	return []builderLine{
		m.labelled(flowFieldFeedOutput, p.T("flows.field_feed_output", "Previous output"), renderComboPills([]string{off, on}, feed), w),
		m.labelled(flowFieldWait, p.T("flows.field_wait", "When it ends"), renderComboPills([]string{auto, human}, wait), w),
	}
}

// groupRow is one heading over a run of fields, with a rule after it so the
// eye finds the next group without reading the words again.
func (m Model) groupRow(head string, w int) builderLine {
	line := "  " + Paint(Live).Bold(true).Render(head) + " "

	// Every rule ends in the same column, so the groups read as one
	// column of the form rather than as four ragged ones.
	if rule := min(w, 104) - lipgloss.Width(line) - 2; rule > 0 {
		line += Paint(Dim).Render(strings.Repeat("─", rule))
	}

	return plainLine(fit(line, w))
}

// labelled is one label and its value, with the cursor's mark when the
// reader is on it.
func (m Model) labelled(field int, label, val string, w int) builderLine {
	mark, lbl := "  ", pad(label, labelWidth, false)

	if m.flows.field == field {
		mark = Paint(Accent).Bold(true).Render("▸ ")
		lbl = Paint(Accent).Bold(true).Render(lbl)
	} else {
		lbl = Paint(Dim).Render(lbl)
	}

	return builderLine{text: fit(mark+lbl+" "+val, w), field: field, phase: noPhase, pick: noPick}
}

// typedValue is a one-line text field: what it holds, with the caret after
// it while it is the field being typed into.
func (m Model) typedValue(field int, val string) string {
	if m.flows.field == field {
		return Paint(Accent).Render(val + "█")
	}

	if val == "" {
		return Paint(Dim).Render(m.opts.Words.T("flows.empty_field", "(empty)"))
	}

	return Text(Primary).Render(val)
}
