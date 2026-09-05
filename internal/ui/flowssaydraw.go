package ui

// What the "describe it" tab draws.

import "strconv"

// sayDial is one of the tab's two dials: what it is called, where it stands,
// and the mark when the keys are aimed at it.
func (m Model) sayDial(on int, label, value, act string, w int) builderLine {
	mark, lbl := "  ", pad(label, labelWidth, false)

	if m.flows.sayFocus == on {
		mark = Paint(Accent).Bold(true).Render("▸ ")
		lbl = Paint(Accent).Bold(true).Render(lbl)
		value += "  " + Paint(Dim).Render(m.opts.Words.T("flows.say_dial_ways", "← → change · [↵] see them all"))
	} else {
		lbl = Paint(Dim).Render(lbl)
	}

	return builderLine{text: fit(mark+lbl+" "+value, w), field: noField, phase: noPhase, pick: noPick, act: act}
}

// sayReplaces warns that a draft would take the place of what is already
// written, which is the one thing this tab does that cannot be undone by
// pressing escape.
func (m Model) sayReplaces(w int) builderLine {
	if len(m.flows.phases) == 0 || !m.flows.isEditing {
		return plainLine("")
	}

	return plainLine(fit("  "+Paint(Warn).Render(m.opts.Words.T("flows.say_replaces",
		"a draft replaces the {n} phases this flow already has",
		about("n", strconv.Itoa(len(m.flows.phases))))), w))
}

// sayRows is the tab: what it is for, the box the sentence goes in, and
// whatever the last attempt had to say.
func (m Model) sayRows(w int, sz boxSizes) []builderLine {
	st := &m.flows
	p := m.opts.Words

	mdls, mdlLabels := m.pickerChoices(flowFieldSayModel)

	out := []builderLine{
		plainLine(""),
		plainLine(fit("  "+Paint(Accent).Bold(true).Render(p.T("flows.say_title",
			"Say what the flow should do")), w)),
		plainLine(fit("  "+Paint(Dim).Render(p.T("flows.say_about",
			"the draft lands in the other tabs to check; nothing is saved until you press Save Flow")), w)),
		m.sayReplaces(w),
		plainLine(""),
		m.sayDial(sayOnEngine, p.T("flows.say_ask_engine", "Ask"),
			renderComboPills(m.engineNames(), m.sayEngineName()), "say_engine", w),
		m.sayDial(sayOnModel, p.T("flows.say_ask_model", "on model"),
			m.dialValue(flowFieldSayModel, mdls, mdlLabels, m.flows.sayModel), "say_model", w),
		plainLine(""),
	}

	out = append(out, m.textBox(flowFieldPrompt, st.say,
		p.T("flows.say_placeholder",
			"e.g. implement, then go round fixing until the tests pass and coverage is over 90%, then a review that stops for me"),
		max(sz.prompt+2, 5), w)...)

	out = append(out, plainLine(""))

	switch {
	case st.saying:
		out = append(out, plainLine(fit("  "+Paint(Live).Render(p.T("flows.say_asking",
			"asking {engine}…", about("engine", m.sayEngineName()))), w)))
	case st.sayNote != "":
		out = append(out, plainLine(fit("  "+Paint(Bad).Render(st.sayNote), w)))
	default:
		out = append(out, builderLine{
			text:  fit("  "+Pill(p.T("flows.btn_draft", "✨ Draft it"), "#FFFFFF", "#581C87")+"  "+Paint(Dim).Render(p.T("flows.draft_same_as", "or press ↵")), w),
			field: noField,
			phase: noPhase,
			pick:  noPick,
			act:   "draft",
		})
	}

	return append(out,
		plainLine(""),
		plainLine(fit("  "+Paint(Dim).Render(p.T("flows.say_ways2",
			"[tab] engine · model · text · [↵] pick, or draft it from the text · [shift+↵] new line · [^V] paste · [^←/^→] tab · [esc] back")), w)),
	)
}
