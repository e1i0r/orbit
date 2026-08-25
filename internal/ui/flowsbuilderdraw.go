package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
)

func (m Model) startCreateFlow() Model {
	m.flows.creating = true
	m.flows.isEditing = false
	m.flows.confirmDiscard = false
	m.flows.confirmDelete = false
	m.flows.field = 0
	m.flows.template = "ninguna"
	m.flows.flowName = ""
	m.flows.phases = []flow.Phase{
		{Name: "1-implement", Engine: "claude", Model: "sonnet", Effort: "default", Thinking: "adaptive", Permissions: []string{"repo"}},
	}
	m.flows.activePhase = 0
	return m
}

func (m Model) flowsBuilderRows(h, w int) []string {
	st := &m.flows
	st.ensurePhase()
	p := m.opts.Words
	title := p.T("flows.builder_create_title", "Flow Designer (Create New Flow)")
	if st.isEditing {
		title = p.T("flows.builder_edit_title", "Flow Designer (Edit: {name})", about("name", st.flowName))
	}
	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(title),
		"  " + Paint(Dim).Render(p.T("flows.builder_subtitle", "configure phases, models, effort, thinking, input chaining and prompts")),
		"",
	}

	// 1. Visual Pipeline Overview
	out = append(out, "  "+Paint(Live).Bold(true).Render(p.T("flows.builder_pipeline_title", "Pipeline (Click on a phase to edit it):")))
	for i, ph := range st.phases {
		prefix := "    ➔ "
		if i == 0 {
			prefix = "    "
		}
		curMark := "○ "
		if i == st.activePhase {
			curMark = "● "
		}
		phaseLabel := p.T("flows.phase_label", "Phase")
		title := fmt.Sprintf("%s%s %d: %s (%s/%s)", curMark, phaseLabel, i+1, ph.Name, ph.Engine, orDef(ph.Model, "default"))
		if ph.FeedOutput {
			title += " " + p.T("flows.feeds_input", "[feeds input]")
		}
		if ph.Wait {
			title += " " + p.T("flows.stops_human", "(stops for human)")
		}

		if i == st.activePhase {
			out = append(out, prefix+Paint(Accent).Bold(true).Render(title))
		} else {
			out = append(out, prefix+Paint(Dim).Render(title))
		}
		if ph.Prompt != "" {
			prmStr := fit(`"`+ph.Prompt+`"`, w-14)
			out = append(out, "       "+Paint(Dim).Render(prmStr))
		}
	}
	out = append(out, "")

	curPh := st.cur()

	renderField := func(fieldIdx int, label string, val string) string {
		mark := "  "
		if st.field == fieldIdx {
			mark = Paint(Accent).Bold(true).Render("▸ ")
		}
		lbl := pad(label, 26, false)
		if st.field == fieldIdx {
			lbl = Paint(Accent).Bold(true).Render(lbl)
		} else {
			lbl = Paint(Dim).Render(lbl)
		}
		return mark + lbl + " " + val
	}

	// 0. Preset Template
	tplPills := renderComboPills([]string{"ninguna", "TDD Cycle", "Security Audit", "Turbo Fix"}, st.template)
	out = append(out, renderField(flowFieldTemplate, p.T("flows.field_template", "0. Template / Preset"), tplPills))

	// 1. Flow Name
	fNameVal := "[" + st.flowName + "_]"
	if st.field != flowFieldName && st.flowName != "" {
		fNameVal = st.flowName
	}
	out = append(out, renderField(flowFieldName, p.T("flows.field_flow_name", "1. Flow Name"), Paint(Accent).Render(fNameVal)))

	// 2. Interactive Phase Tabs
	var tabPills []string
	for i, ph := range st.phases {
		label := fmt.Sprintf("%d.%s", i+1, ph.Name)
		if i == st.activePhase {
			tabPills = append(tabPills, Paint(Sel).Bold(true).Render(" ● "+label+" "))
		} else {
			tabPills = append(tabPills, Paint(Dim).Render(" "+label+" "))
		}
	}
	out = append(out, renderField(flowFieldPhaseSelect, p.T("flows.field_editing_phase", "Editing Phase"), strings.Join(tabPills, " ")))

	// 3. Phase Name
	pNameVal := "[" + curPh.Name + "_]"
	if st.field != flowFieldPhaseName && curPh.Name != "" {
		pNameVal = curPh.Name
	}
	out = append(out, renderField(flowFieldPhaseName, p.T("flows.field_phase_name", "2. Phase {n} Name", about("n", strconv.Itoa(st.activePhase+1))), Paint(Accent).Render(pNameVal)))

	// 4. Engine
	engPills := renderComboPills([]string{"claude", "codex", "opencode"}, orDef(curPh.Engine, "claude"))
	out = append(out, renderField(flowFieldEngine, p.T("flows.field_engine", "3. AI Engine"), engPills))

	// 5. Model
	mdlPills := renderComboPills([]string{"sonnet", "opus", "haiku", "default"}, orDef(curPh.Model, "default"))
	out = append(out, renderField(flowFieldModel, p.T("flows.field_model", "4. Model"), mdlPills))

	// 6. Effort
	effPills := renderComboPills([]string{"default", "low", "medium", "high", "xhigh", "max"}, orDef(curPh.Effort, "default"))
	out = append(out, renderField(flowFieldEffort, p.T("flows.field_effort", "5. Effort"), effPills))

	// 7. Thinking Mode
	thkPills := renderComboPills([]string{"adaptive", "on", "off"}, orDef(curPh.Thinking, "adaptive"))
	out = append(out, renderField(flowFieldThinking, p.T("flows.field_thinking", "6. Thinking Mode"), thkPills))

	// 8. Feed Output
	feedVal := "off"
	if curPh.FeedOutput {
		feedVal = "on"
	}
	feedPills := renderComboPills([]string{"off", "on"}, feedVal)
	out = append(out, renderField(flowFieldFeedOutput, p.T("flows.field_feed_output", "7. Feed Previous Output"), feedPills))

	// 9. Control
	waitVal := "auto"
	if curPh.Wait {
		waitVal = "wait (humano)"
	}
	waitPills := renderComboPills([]string{"auto", "wait (humano)"}, waitVal)
	out = append(out, renderField(flowFieldWait, p.T("flows.field_wait", "8. Phase Control"), waitPills))

	// 10. Prompt
	prmHdr := renderField(flowFieldPrompt, p.T("flows.field_prompt", "9. Prompt / Instructions"), "")
	prmHdr += " " + Pill(p.T("flows.btn_paste", "📋 Paste"), "#FFFFFF", "#0C4A6E") + " " + Pill(p.T("flows.btn_autogen", "✨ Autogenerate"), "#FFFFFF", "#581C87") + " " + Pill(p.T("flows.btn_clear", "🗑 Clear"), "#FFFFFF", "#374151")
	out = append(out, fit(prmHdr, w))

	boxWidth := w - 6
	if boxWidth > 80 {
		boxWidth = 80
	}
	if boxWidth < 36 {
		boxWidth = 36
	}

	content := curPh.Prompt
	if st.field == flowFieldPrompt {
		content += "_"
	}
	wrapped := wrapPromptText(content, boxWidth-4)
	if len(wrapped) == 0 {
		wrapped = []string{p.T("flows.prompt_placeholder", "(type instructions here or click ✨ Autogenerate)...")}
	}
	if len(wrapped) > 4 {
		wrapped = wrapped[:4]
	}

	boxBorderColor := Dim
	if st.field == flowFieldPrompt {
		boxBorderColor = Accent
	}

	out = append(out, Paint(boxBorderColor).Render("    ┌"+strings.Repeat("─", boxWidth-2)+"┐"))
	for _, l := range wrapped {
		linePad := pad(l, boxWidth-4, false)
		txtStyle := Paint(Dim)
		if st.field == flowFieldPrompt {
			txtStyle = Paint(Accent)
		}
		out = append(out, "    "+Paint(boxBorderColor).Render("│ ")+txtStyle.Render(linePad)+Paint(boxBorderColor).Render(" │"))
	}
	out = append(out, Paint(boxBorderColor).Render("    └"+strings.Repeat("─", boxWidth-2)+"┘"))

	// Actions
	out = append(out, "")
	btnMark1, btnMark2, btnMark3 := "  ", "  ", "  "
	if st.field == flowFieldAddPhase {
		btnMark1 = Paint(Accent).Bold(true).Render("▸ ")
	}
	if st.field == flowFieldDelPhase {
		btnMark2 = Paint(Accent).Bold(true).Render("▸ ")
	}
	if st.field == flowFieldSave {
		btnMark3 = Paint(Accent).Bold(true).Render("▸ ")
	}
	btn1 := btnMark1 + Pill(p.T("flows.btn_add_phase", "+ Add Phase"), "#FFFFFF", "#0C4A6E")
	btn2 := btnMark2 + Pill(p.T("flows.btn_del_phase", "🗑 Delete Phase"), "#FFFFFF", "#7F1D1D")
	btn3 := btnMark3 + Pill(p.T("flows.btn_save_flow", "✔ Save Flow"), "#FFFFFF", "#14532D")
	out = append(out, "  "+btn1+"    "+btn2+"    "+btn3, "")

	waysOut := p.T("flows.ways_out_form", "[tab] field · [enter] action · {back} back",
		about("back", m.keys.Back.Help().Key))
	out = append(out, fit("  "+Paint(Dim).Render(waysOut), w))
	return fill(out, h)
}
