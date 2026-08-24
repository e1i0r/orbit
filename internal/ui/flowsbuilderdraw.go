package ui

import (
	"fmt"
	"strings"
)

func (m Model) flowsBuilderRows(h, w int) []string {
	st := &m.flows
	st.ensurePhase()
	p := m.opts.Words
	title := "Diseñador de Flujos (Crear Nuevo Flujo)"
	if st.isEditing {
		title = fmt.Sprintf("Diseñador de Flujos (Editar: %s)", st.flowName)
	}
	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(title),
		"  " + Paint(Dim).Render("configura las fases, modelos, esfuerzo, thinking, input chaining y prompts"),
		"",
	}

	// 1. Visual Pipeline Overview
	out = append(out, "  "+Paint(Live).Bold(true).Render("Pipeline del Ciclo (Haz clic en una fase para editarla):"))
	for i, ph := range st.phases {
		prefix := "    ➔ "
		if i == 0 {
			prefix = "    "
		}
		curMark := "○ "
		if i == st.activePhase {
			curMark = "● "
		}
		title := fmt.Sprintf("%sFase %d: %s (%s/%s)", curMark, i+1, ph.Name, ph.Engine, orDef(ph.Model, "default"))
		if ph.FeedOutput {
			title += " [feeds input]"
		}
		if ph.Wait {
			title += " (stops for human)"
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
	out = append(out, renderField(flowFieldTemplate, "0. Plantilla / Preset", tplPills))

	// 1. Flow Name
	fNameVal := "[" + st.flowName + "_]"
	if st.field != flowFieldName && st.flowName != "" {
		fNameVal = st.flowName
	}
	out = append(out, renderField(flowFieldName, "1. Nombre del Flujo", Paint(Accent).Render(fNameVal)))

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
	out = append(out, renderField(flowFieldPhaseSelect, "Fase en edición", strings.Join(tabPills, " ")))

	// 3. Phase Name
	pNameVal := "[" + curPh.Name + "_]"
	if st.field != flowFieldPhaseName && curPh.Name != "" {
		pNameVal = curPh.Name
	}
	out = append(out, renderField(flowFieldPhaseName, fmt.Sprintf("2. Nombre Fase %d", st.activePhase+1), Paint(Accent).Render(pNameVal)))

	// 4. Engine
	engPills := renderComboPills([]string{"claude", "codex", "opencode"}, orDef(curPh.Engine, "claude"))
	out = append(out, renderField(flowFieldEngine, "3. Motor IA", engPills))

	// 5. Model
	mdlPills := renderComboPills([]string{"sonnet", "opus", "haiku", "default"}, orDef(curPh.Model, "default"))
	out = append(out, renderField(flowFieldModel, "4. Modelo", mdlPills))

	// 6. Effort
	effPills := renderComboPills([]string{"default", "low", "medium", "high", "xhigh", "max"}, orDef(curPh.Effort, "default"))
	out = append(out, renderField(flowFieldEffort, "5. Esfuerzo", effPills))

	// 7. Thinking Mode
	thkPills := renderComboPills([]string{"adaptive", "on", "off"}, orDef(curPh.Thinking, "adaptive"))
	out = append(out, renderField(flowFieldThinking, "6. Modo Thinking", thkPills))

	// 8. Feed Output
	feedVal := "off"
	if curPh.FeedOutput {
		feedVal = "on"
	}
	feedPills := renderComboPills([]string{"off", "on"}, feedVal)
	out = append(out, renderField(flowFieldFeedOutput, "7. Alimentar Output Anterior", feedPills))

	// 9. Control
	waitVal := "auto"
	if curPh.Wait {
		waitVal = "wait (humano)"
	}
	waitPills := renderComboPills([]string{"auto", "wait (humano)"}, waitVal)
	out = append(out, renderField(flowFieldWait, "8. Control de Fase", waitPills))

	// 10. Prompt
	prmHdr := renderField(flowFieldPrompt, "9. Prompt / Instrucciones", "")
	prmHdr += " " + Pill("📋 Pegar", "#FFFFFF", "#0C4A6E") + " " + Pill("✨ Autogenerar", "#FFFFFF", "#581C87") + " " + Pill("🗑 Limpiar", "#FFFFFF", "#374151")
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
		wrapped = []string{"(escribe aquí las instrucciones o pulsa ✨ Autogenerar)..."}
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
	btn1 := btnMark1 + Pill("+ Añadir Fase", "#FFFFFF", "#0C4A6E")
	btn2 := btnMark2 + Pill("🗑 Borrar Fase", "#FFFFFF", "#7F1D1D")
	btn3 := btnMark3 + Pill("✔ Guardar Flujo", "#FFFFFF", "#14532D")
	out = append(out, "  "+btn1+"    "+btn2+"    "+btn3, "")

	waysOut := p.T("flows.ways_out_form", "[tab] field · [enter] action · {back} back",
		about("back", m.keys.Back.Help().Key))
	out = append(out, fit("  "+Paint(Dim).Render(waysOut), w))
	return fill(out, h)
}
