package flows

// What the "describe it" tab draws.

import (
	"strconv"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// sayDial is one of the tab's two dials: what it is called, where it stands,
// and the mark when the keys are aimed at it.
func (s State) sayDial(on int, label, value, act string, w int, e Env) builderLine {
	mark, lbl := "  ", cells.Pad(label, labelWidth, false)

	if s.sayFocus == on {
		mark = theme.Paint(theme.Accent).Bold(true).Render("▸ ")
		lbl = theme.Paint(theme.Accent).Bold(true).Render(lbl)
		value += "  " + theme.Paint(theme.Dim).Render(e.Words.T("flows.say_dial_ways", "← → change · [↵] see them all"))
	} else {
		lbl = theme.Paint(theme.Dim).Render(lbl)
	}

	return builderLine{text: cells.Fit(mark+lbl+" "+value, w), field: noField, phase: noPhase, pick: noPick, act: act}
}

// sayReplaces warns that a draft would take the place of what is already
// written, which is the one thing this tab does that cannot be undone by
// pressing escape.
func (s State) sayReplaces(w int, e Env) builderLine {
	if len(s.phases) == 0 || !s.isEditing {
		return plainLine("")
	}

	return plainLine(cells.Fit("  "+theme.Paint(theme.Warn).Render(e.Words.T("flows.say_replaces",
		"a draft replaces the {n} phases this flow already has",
		about("n", strconv.Itoa(len(s.phases))))), w))
}

// sayRows is the tab: what it is for, the box the sentence goes in, and
// whatever the last attempt had to say.
func (s State) sayRows(w int, sz boxSizes, e Env) []builderLine {
	p := e.Words

	mdls, mdlLabels := s.pickerChoices(flowFieldSayModel, e)

	out := []builderLine{
		plainLine(""),
		plainLine(cells.Fit("  "+theme.Paint(theme.Accent).Bold(true).Render(p.T("flows.say_title",
			"Say what the flow should do")), w)),
		plainLine(cells.Fit("  "+theme.Paint(theme.Dim).Render(p.T("flows.say_about",
			"the draft lands in the other tabs to check; nothing is saved until you press Save Flow")), w)),
		s.sayReplaces(w, e),
		plainLine(""),
		s.sayDial(sayOnEngine, p.T("flows.say_ask_engine", "Ask"),
			renderComboPills(e.Engines(), s.sayEngineName(e)), "say_engine", w, e),
		s.sayDial(sayOnModel, p.T("flows.say_ask_model", "on model"),
			s.dialValue(flowFieldSayModel, mdls, mdlLabels, s.sayModel, e), "say_model", w, e),
		plainLine(""),
	}

	out = append(out, s.textBox(flowFieldPrompt, s.say,
		p.T("flows.say_placeholder",
			"e.g. implement, then go round fixing until the tests pass and coverage is over 90%, then a review that stops for me"),
		max(sz.prompt+2, 5), w)...)

	out = append(out, plainLine(""))

	switch {
	case s.saying:
		out = append(out, plainLine(cells.Fit("  "+e.Spinner(theme.Live)+theme.Paint(theme.Live).Render(p.T("flows.say_asking2",
			"asking {engine} — {secs} · [esc] stop waiting",
			about("engine", s.sayEngineName(e)), about("secs", s.waitedFor(e).String()))), w)))
	case s.sayNote != "":
		out = append(out, plainLine(cells.Fit("  "+theme.Paint(theme.Bad).Render(s.sayNote), w)))
	default:
		button := theme.Pill(p.T("flows.btn_draft", "✨ Draft it"), "#FFFFFF", "#581C87")
		also := theme.Paint(theme.Dim).Render(p.T("flows.draft_same_as", "or press ↵"))

		out = append(out, builderLine{
			text:  cells.Fit("  "+button+"  "+also, w),
			field: noField,
			phase: noPhase,
			pick:  noPick,
			act:   "draft",
		})
	}

	return append(out,
		plainLine(""),
		plainLine(cells.Fit("  "+theme.Paint(theme.Dim).Render(p.T("flows.say_ways2",
			"[tab] engine · model · text · [↵] pick, or draft it from the text · [shift+↵] new line · [^V] paste · [^←/^→] tab · [esc] back")), w)),
	)
}
