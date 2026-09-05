package ui

// The two fields of the designer that hold a paragraph: what the phase is
// told to do, and what has to pass for a loop to stop.
//
// Both are boxes rather than lines. A phase's instructions are the part of a
// flow somebody actually writes, and a field that shows one line of them is
// a field that sends the reader back to editing the JSON by hand — which is
// what this screen is for not doing.

import "strings"

// promptRows is how tall the instruction box is, and checkRows the box of
// checks under a loop. What is being typed is the end of the paragraph, so a
// longer one shows its last lines rather than its first.
//
// Six and four rather than more: the whole form has to fit a terminal, and
// every row of box is a row of the pipeline above it that scrolls away.
const (
	promptRows = 6
	checkRows  = 4
	descRows   = 3
)

// builderPromptRows is the instructions field: its label, the three pills
// that fill it, and the box itself.
func (m Model) builderPromptRows(w int, sz boxSizes) []builderLine {
	st := &m.flows
	p := m.opts.Words

	head := m.labelled(flowFieldPrompt, p.T("flows.field_prompt", "Instructions"), "", w)
	head.head = true
	head.text = fit(strings.TrimRight(head.text, " ")+" "+
		Pill(p.T("flows.btn_paste", "📋 Paste"), "#FFFFFF", "#0C4A6E")+" "+
		Pill(p.T("flows.btn_autogen", "✨ Autogenerate"), "#FFFFFF", "#581C87")+" "+
		Pill(p.T("flows.btn_clear", "🗑 Clear"), "#FFFFFF", "#374151"), w)

	out := []builderLine{head}

	return append(out, m.textBox(flowFieldPrompt, st.edited().Prompt,
		p.T("flows.prompt_placeholder", "(type instructions here or click ✨ Autogenerate)..."), sz.prompt, w)...)
}

// textBox is a paragraph in a frame, and every row of it belongs to the
// field it is drawn for, so a click anywhere inside lands on that field.
func (m Model) textBox(field int, content, placeholder string, rows, w int) []builderLine {
	// As wide as the window allows, up to a line length that is still
	// comfortable to read back: what somebody types here is a paragraph, and
	// eighty columns of it on a hundred-and-thirty column terminal was a
	// third of the screen left empty beside the field being written in.
	boxWidth := min(max(w-6, 36), 104)

	shown := content
	if m.flows.field == field {
		shown += "_"
	}

	wrapped := wrapPromptText(shown, boxWidth-4)

	ghost := len(wrapped) == 0
	if ghost {
		wrapped = []string{placeholder}
	}

	// The tail and not the head: the caret is at the end of what has been
	// typed, and a box that showed the first lines would scroll away from
	// the reader as they wrote.
	if len(wrapped) > rows {
		wrapped = wrapped[len(wrapped)-rows:]
	}

	ink, edge := Paint(Dim), Dim
	if m.flows.field == field {
		edge = Accent

		if !ghost {
			ink = Paint(Accent)
		}
	}

	line := func(text string) builderLine {
		return builderLine{text: fit(text, w), field: field, phase: noPhase, pick: noPick}
	}

	out := []builderLine{line("    " + Paint(edge).Render("┌"+strings.Repeat("─", boxWidth-2)+"┐"))}
	for _, l := range wrapped {
		out = append(out, line("    "+Paint(edge).Render("│ ")+ink.Render(pad(l, boxWidth-4, false))+Paint(edge).Render(" │")))
	}

	return append(out, line("    "+Paint(edge).Render("└"+strings.Repeat("─", boxWidth-2)+"┘")))
}

// loopFieldRows is the two fields a repeating phase has: how many turns it
// may take, and what has to pass for it to stop.
func (m Model) loopFieldRows(w int, sz boxSizes) []builderLine {
	st := &m.flows
	p := m.opts.Words

	out := []builderLine{
		m.labelled(flowFieldLoopTurns, "  ├ "+p.T("flows.field_turns", "turns at most"),
			Paint(Accent).Render(st.loopTurnsText())+"  "+
				Paint(Dim).Render(p.T("flows.turns_hint", "← → to change")), w),
		m.labelled(flowFieldLoopUntil, "  └ "+p.T("flows.field_until", "stops when all pass"),
			Paint(Dim).Render(p.T("flows.until_hint", "one per line — name: command")), w),
	}

	return append(out, m.textBox(flowFieldLoopUntil, st.loopChecksText(),
		p.T("flows.until_placeholder", "tests: go test ./..."), sz.checks, w)...)
}

// builderActions is the three buttons, the sentence explaining the field the
// reader is on, and the ways out.
func (m Model) builderActions(w int) []builderLine {
	p := m.opts.Words

	if m.flows.readOnly {
		return []builderLine{
			plainLine(""),
			plainLine("  " + Pill(" ↵ "+p.T("flows.btn_return", "Return")+" ", "#FFFFFF", "#2563EB")),
			plainLine(""),
			plainLine(fit("  "+Paint(Dim).Render(p.T("flows.ways_out_preview",
				"[←/→ / tab] inspect phase · [enter / esc] return")), w)),
		}
	}

	mark := func(field int) string {
		if m.flows.field == field {
			return Paint(Accent).Bold(true).Render("▸ ")
		}

		return "  "
	}

	buttons := "  " + mark(flowFieldAddPhase) + Pill(p.T("flows.btn_add_phase", "+ Add Phase"), "#FFFFFF", "#0C4A6E") +
		"    " + mark(flowFieldDelPhase) + Pill(p.T("flows.btn_del_phase", "🗑 Delete Phase"), "#FFFFFF", "#7F1D1D") +
		"    " + mark(flowFieldSave) + Pill(p.T("flows.btn_save_flow", "✔ Save Flow"), "#FFFFFF", "#14532D")

	return []builderLine{
		{text: fit(buttons, w), field: flowFieldAddPhase, phase: noPhase, pick: noPick},
		plainLine(fit("  "+Paint(Live).Render(m.fieldHint()), w)),
		plainLine(fit("  "+Paint(Dim).Render(p.T("flows.ways_out_form",
			"[tab] next field · [↑↓] move · [←→] change · [enter] do it · [shift+↵] new line · {back} back",
			about("back", m.keys.Back.Help().Key))), w)),
	}
}
