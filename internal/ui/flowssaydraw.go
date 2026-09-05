package ui

// What the "describe it" tab draws.

// sayRows is the tab: what it is for, the box the sentence goes in, and
// whatever the last attempt had to say.
func (m Model) sayRows(w int, sz boxSizes) []builderLine {
	st := &m.flows
	p := m.opts.Words

	out := []builderLine{
		plainLine(""),
		plainLine(fit("  "+Paint(Accent).Bold(true).Render(p.T("flows.say_title",
			"Say what the flow should do"))+"  "+
			Paint(Dim).Render(p.T("flows.say_engine", "asks {engine}", about("engine", m.dialEngine("")))), w)),
		plainLine(fit("  "+Paint(Dim).Render(p.T("flows.say_about",
			"the draft lands in the other two tabs for you to check; nothing is saved until you press Save Flow")), w)),
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
			"asking {engine}… this takes as long as one of its runs", about("engine", m.dialEngine("")))), w)))
	case st.sayNote != "":
		out = append(out, plainLine(fit("  "+Paint(Bad).Render(st.sayNote), w)))
	default:
		out = append(out, plainLine(fit("  "+Pill(p.T("flows.btn_draft", "✨ Draft it"), "#FFFFFF", "#581C87"), w)))
	}

	return append(out,
		plainLine(""),
		plainLine(fit("  "+Paint(Dim).Render(p.T("flows.say_ways",
			"[↵] draft it · [shift+↵] new line · [^V] paste · [^←/^→] tab · [esc] back")), w)),
	)
}
