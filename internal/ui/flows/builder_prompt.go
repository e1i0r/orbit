package flows

// The two fields of the designer that hold a paragraph: what the phase is
// told to do, and what has to pass for a loop to stop.
//
// Both are boxes rather than lines. A phase's instructions are the part of a
// flow somebody actually writes, and a field that shows one line of them is
// a field that sends the reader back to editing the JSON by hand — which is
// what this screen is for not doing.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

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
func (s State) builderPromptRows(w int, sz boxSizes, e Env) []builderLine {
	p := e.Words

	head := s.labelled(flowFieldPrompt, p.T("flows.field_prompt", "Instructions"), "", w)
	head.head = true
	head.text = cells.Fit(strings.TrimRight(head.text, " ")+" "+
		theme.Pill(p.T("flows.btn_paste", "📋 Paste"), theme.PillInk, theme.PillEdit)+" "+
		theme.Pill(p.T("flows.btn_autogen", "✨ Autogenerate"), theme.PillInk, theme.PillDraft)+" "+
		theme.Pill(p.T("flows.btn_clear", "🗑 Clear"), theme.PillInk, theme.PillClear), w)

	out := []builderLine{head}

	return append(out, s.textBox(flowFieldPrompt, s.edited().Prompt,
		p.T("flows.prompt_placeholder", "(type instructions here or click ✨ Autogenerate)..."), sz.prompt, w)...)
}

// textBox is a paragraph in a frame, and every row of it belongs to the
// field it is drawn for, so a click anywhere inside lands on that field.
func (s State) textBox(field int, content, placeholder string, rows, w int) []builderLine {
	// As wide as the window allows, up to a line length that is still
	// comfortable to read back: what somebody types here is a paragraph, and
	// eighty columns of it on a hundred-and-thirty column terminal was a
	// third of the screen left empty beside the field being written in.
	boxWidth := min(max(w-6, 36), 104)

	shown := content
	if s.field == field {
		shown += "_"
	}

	wrapped := cells.WrapKeeping(shown, boxWidth-4)

	// The placeholder is folded like anything else in the box: it is a
	// sentence saying what to write here, and one cut off at the edge with
	// an ellipsis is a sentence nobody finishes reading.
	ghost := len(wrapped) == 0
	if ghost {
		wrapped = cells.WrapKeeping(placeholder, boxWidth-4)
	}

	// The tail and not the head: the caret is at the end of what has been
	// typed, and a box that showed the first lines would scroll away from
	// the reader as they wrote.
	if len(wrapped) > rows {
		wrapped = wrapped[len(wrapped)-rows:]
	}

	ink, edge := theme.Paint(theme.Dim), theme.Dim
	if s.field == field {
		edge = theme.Accent

		if !ghost {
			ink = theme.Paint(theme.Accent)
		}
	}

	line := func(text string) builderLine {
		return builderLine{text: cells.Fit(text, w), field: field, phase: noPhase, pick: noPick}
	}

	out := []builderLine{line("    " + theme.Paint(edge).Render("┌"+strings.Repeat("─", boxWidth-2)+"┐"))}
	for _, l := range wrapped {
		out = append(out, line("    "+theme.Paint(edge).Render("│ ")+ink.Render(cells.Pad(l, boxWidth-4, false))+theme.Paint(edge).Render(" │")))
	}

	return append(out, line("    "+theme.Paint(edge).Render("└"+strings.Repeat("─", boxWidth-2)+"┘")))
}

// loopFieldRows is the two fields a repeating phase has: how many turns it
// may take, and what has to pass for it to stop.
func (s State) loopFieldRows(w int, sz boxSizes, e Env) []builderLine {
	p := e.Words

	out := []builderLine{
		s.labelled(flowFieldLoopTurns, "  ├ "+p.T("flows.field_turns", "turns at most"),
			theme.Paint(theme.Accent).Render(s.loopTurnsText())+"  "+
				theme.Paint(theme.Dim).Render(p.T("flows.turns_hint", "← → to change")), w),
		s.labelled(flowFieldLoopUntil, "  └ "+p.T("flows.field_until", "stops when all pass"),
			theme.Paint(theme.Dim).Render(p.T("flows.until_hint", "one per line — name: command")), w),
	}

	return append(out, s.textBox(flowFieldLoopUntil, s.loopChecksText(),
		p.T("flows.until_placeholder", "tests: go test ./..."), sz.checks, w)...)
}

// builderActions is the three buttons, the sentence explaining the field the
// reader is on, and the ways out.
func (s State) builderActions(w int, e Env) []builderLine {
	p := e.Words

	if s.readOnly {
		return []builderLine{
			plainLine(""),
			plainLine("  " + theme.Pill(" ↵ "+p.T("flows.btn_return", "Return")+" ", theme.PillInk, theme.PillReturn)),
			plainLine(""),
			plainLine(cells.Fit("  "+theme.Paint(theme.Dim).Render(p.T("flows.ways_out_preview",
				"[←/→ / tab] inspect phase · [enter / esc] return")), w)),
		}
	}

	mark := func(field int) string {
		if s.field == field {
			return theme.Paint(theme.Accent).Bold(true).Render("▸ ")
		}

		return "  "
	}

	buttons := "  " + mark(flowFieldAddPhase) + theme.Pill(p.T("flows.btn_add_phase", "+ Add Phase"), theme.PillInk, theme.PillEdit) +
		"    " + mark(flowFieldDelPhase) + theme.Pill(p.T("flows.btn_del_phase", "🗑 Delete Phase"), theme.PillInk, theme.PillDelete) +
		"    " + mark(flowFieldSave) + theme.Pill(p.T("flows.btn_save_flow", "✔ Save Flow"), theme.PillInk, theme.PillSave)

	return []builderLine{
		{text: cells.Fit(buttons, w), field: flowFieldAddPhase, phase: noPhase, pick: noPick},
		plainLine(cells.Fit("  "+theme.Paint(theme.Live).Render(s.fieldHint(e)), w)),
		plainLine(cells.Fit("  "+theme.Paint(theme.Dim).Render(p.T("flows.ways_out_form",
			"[tab] next field · [↑↓] move · [←→] change · [enter] do it · [shift+↵] new line · {back} back",
			about("back", e.Keys.Back.Help().Key))), w)),
	}
}
