package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
)

func (m Model) flowsBuilderRows(h, w int) []string {
	st := &m.flows
	st.ensurePhase()
	p := m.opts.Words
	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render("Diseñador de Flujos (Crear Ciclo / Pipeline)"),
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
	tabLine := "  " + Paint(Live).Render("Fase en edición: ") + strings.Join(tabPills, " ")
	out = append(out, fit(tabLine, w), "")

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

	curPh := st.cur()

	// 0. Preset Template
	tplPills := renderComboPills([]string{"ninguna", "TDD Cycle", "Security Audit", "Turbo Fix"}, st.template)
	out = append(out, renderField(flowFieldTemplate, "0. Plantilla / Preset", tplPills))

	// 1. Flow Name
	fNameVal := "[" + st.flowName + "_]"
	if st.field != flowFieldName && st.flowName != "" {
		fNameVal = st.flowName
	}
	out = append(out, renderField(flowFieldName, "1. Nombre del Flujo", Paint(Accent).Render(fNameVal)))

	// 2. Phase Name
	pNameVal := "[" + curPh.Name + "_]"
	if st.field != flowFieldPhaseName && curPh.Name != "" {
		pNameVal = curPh.Name
	}
	out = append(out, renderField(flowFieldPhaseName, fmt.Sprintf("2. Nombre Fase %d", st.activePhase+1), Paint(Accent).Render(pNameVal)))

	// 3. Engine
	engPills := renderComboPills([]string{"claude", "codex", "opencode"}, orDef(curPh.Engine, "claude"))
	out = append(out, renderField(flowFieldEngine, "3. Motor IA", engPills))

	// 4. Model
	mdlPills := renderComboPills([]string{"sonnet", "opus", "haiku", "default"}, orDef(curPh.Model, "default"))
	out = append(out, renderField(flowFieldModel, "4. Modelo", mdlPills))

	// 5. Effort
	effPills := renderComboPills([]string{"default", "low", "medium", "high", "xhigh", "max"}, orDef(curPh.Effort, "default"))
	out = append(out, renderField(flowFieldEffort, "5. Esfuerzo", effPills))

	// 6. Thinking Mode
	thkPills := renderComboPills([]string{"adaptive", "on", "off"}, orDef(curPh.Thinking, "adaptive"))
	out = append(out, renderField(flowFieldThinking, "6. Modo Thinking", thkPills))

	// 7. Feed Output
	feedVal := "off"
	if curPh.FeedOutput {
		feedVal = "on"
	}
	feedPills := renderComboPills([]string{"off", "on"}, feedVal)
	out = append(out, renderField(flowFieldFeedOutput, "7. Alimentar Output Anterior", feedPills))

	// 8. Control
	waitVal := "auto"
	if curPh.Wait {
		waitVal = "wait (humano)"
	}
	waitPills := renderComboPills([]string{"auto", "wait (humano)"}, waitVal)
	out = append(out, renderField(flowFieldWait, "8. Control de Fase", waitPills))

	// 9. Prompt
	prmVal := "[" + curPh.Prompt + "_]"
	if st.field != flowFieldPrompt && curPh.Prompt != "" {
		prmVal = `"` + curPh.Prompt + `"`
	}
	out = append(out, renderField(flowFieldPrompt, "9. Prompt / Instrucciones", Paint(Accent).Render(prmVal)))

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

func renderComboPills(options []string, current string) string {
	var views []string
	for _, opt := range options {
		if opt == current {
			views = append(views, Paint(Sel).Render(" "+opt+" "))
		} else {
			views = append(views, Paint(Dim).Render(opt))
		}
	}
	return strings.Join(views, " ")
}

func nextOption(options []string, current string, delta int) string {
	if len(options) == 0 {
		return current
	}
	idx := 0
	for i, opt := range options {
		if opt == current {
			idx = i
			break
		}
	}
	nextIdx := (idx + delta) % len(options)
	if nextIdx < 0 {
		nextIdx += len(options)
	}
	return options[nextIdx]
}

func (m Model) hitFlows(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}
	if !m.flows.creating {
		if line == 4 {
			return Target{Kind: TargetFlowItem, Field: "create"}
		}
		descriptors := flow.List(m.opts.Flows)
		curLine := 6
		for i, d := range descriptors {
			fl, _ := flow.Resolve(m.opts.Flows, d.Name)
			phaseCount := len(fl.Phases)
			if line >= curLine && line <= curLine+phaseCount {
				m.flows.sel = i
				if d.Origin != flow.OriginBuiltin && x >= 32 {
					return Target{Kind: TargetFlowItem, Field: "delete", ID: d.Name}
				}
				return Target{Kind: TargetFlowItem, Field: "edit", ID: d.Name}
			}
			curLine += 1 + phaseCount + 1
		}
		return Target{}
	}

	st := &m.flows
	// Count overview lines
	overviewLines := 4
	for _, ph := range st.phases {
		overviewLines++
		if ph.Prompt != "" {
			overviewLines++
		}
	}
	overviewLines += 2 // empty line + tab line

	if line >= 4 && line < 4+len(st.phases)*2 {
		idx := (line - 4) / 2
		if idx >= 0 && idx < len(st.phases) {
			return Target{Kind: TargetFlowItem, Field: "select_phase", Phase: idx}
		}
	}

	fLine := line - overviewLines
	switch fLine {
	case 0:
		return Target{Kind: TargetFlowItem, Phase: flowFieldTemplate}
	case 1:
		return Target{Kind: TargetFlowItem, Phase: flowFieldName}
	case 2:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPhaseName}
	case 3:
		return Target{Kind: TargetFlowItem, Phase: flowFieldEngine}
	case 4:
		return Target{Kind: TargetFlowItem, Phase: flowFieldModel}
	case 5:
		return Target{Kind: TargetFlowItem, Phase: flowFieldEffort}
	case 6:
		return Target{Kind: TargetFlowItem, Phase: flowFieldThinking}
	case 7:
		return Target{Kind: TargetFlowItem, Phase: flowFieldFeedOutput}
	case 8:
		return Target{Kind: TargetFlowItem, Phase: flowFieldWait}
	case 9:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPrompt}
	case 11:
		if x < 25 {
			return Target{Kind: TargetFlowItem, Field: "add_phase"}
		}
		if x < 45 {
			return Target{Kind: TargetFlowItem, Field: "del_phase"}
		}
		return Target{Kind: TargetFlowItem, Field: "save"}
	}
	return Target{}
}
