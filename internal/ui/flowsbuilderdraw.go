package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
)

func (m Model) flowsBuilderRows(h, w int) []string {
	st := &m.flows
	p := m.opts.Words
	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render("Diseñador de Flujos (Crear Ciclo / Pipeline)"),
		"  " + Paint(Dim).Render("configura las fases, modelos, esfuerzo, thinking, input chaining y prompts"),
		"",
	}

	if len(st.phases) > 0 {
		var phaseNames []string
		for i, ph := range st.phases {
			phaseNames = append(phaseNames, fmt.Sprintf("%d.%s(%s)", i+1, ph.Name, ph.Model))
		}
		out = append(out, "  "+Paint(Live).Render("Ciclo actual: ")+Paint(Accent).Render(strings.Join(phaseNames, " ➔ ")), "")
	}

	renderField := func(fieldIdx int, label string, val string, isCombo bool) string {
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
	out = append(out, renderField(flowFieldTemplate, "0. Plantilla / Preset", tplPills, true))

	// 1. Flow Name
	fNameVal := "[" + st.flowName + "_]"
	if st.field != flowFieldName && st.flowName != "" {
		fNameVal = st.flowName
	}
	out = append(out, renderField(flowFieldName, "1. Nombre del Flujo", Paint(Accent).Render(fNameVal), false))

	// 2. Phase Name
	pNameVal := "[" + st.phaseName + "_]"
	if st.field != flowFieldPhaseName && st.phaseName != "" {
		pNameVal = st.phaseName
	}
	out = append(out, renderField(flowFieldPhaseName, fmt.Sprintf("2. Nombre Fase %d", len(st.phases)+1), Paint(Accent).Render(pNameVal), false))

	// 3. Engine
	engPills := renderComboPills([]string{"claude", "codex", "opencode"}, st.engine)
	out = append(out, renderField(flowFieldEngine, "3. Motor IA", engPills, true))

	// 4. Model
	mdlPills := renderComboPills([]string{"sonnet", "opus", "haiku", "default"}, st.model)
	out = append(out, renderField(flowFieldModel, "4. Modelo", mdlPills, true))

	// 5. Effort
	effPills := renderComboPills([]string{"default", "low", "medium", "high", "xhigh", "max"}, st.effort)
	out = append(out, renderField(flowFieldEffort, "5. Esfuerzo", effPills, true))

	// 6. Thinking Mode
	thkPills := renderComboPills([]string{"adaptive", "on", "off"}, st.thinking)
	out = append(out, renderField(flowFieldThinking, "6. Modo Thinking", thkPills, true))

	// 7. Feed Output
	feedVal := "off"
	if st.feedOutput {
		feedVal = "on"
	}
	feedPills := renderComboPills([]string{"off", "on"}, feedVal)
	out = append(out, renderField(flowFieldFeedOutput, "7. Alimentar Output Anterior", feedPills, true))

	// 8. Control
	waitVal := "auto"
	if st.wait {
		waitVal = "wait (humano)"
	}
	waitPills := renderComboPills([]string{"auto", "wait (humano)"}, waitVal)
	out = append(out, renderField(flowFieldWait, "8. Control de Fase", waitPills, true))

	// 9. Prompt
	prmVal := "[" + st.prompt + "_]"
	if st.field != flowFieldPrompt && st.prompt != "" {
		prmVal = `"` + st.prompt + `"`
	}
	out = append(out, renderField(flowFieldPrompt, "9. Prompt / Instrucciones", Paint(Accent).Render(prmVal), false))

	// Actions
	out = append(out, "")
	btnMark1, btnMark2 := "  ", "  "
	if st.field == flowFieldAddPhase {
		btnMark1 = Paint(Accent).Bold(true).Render("▸ ")
	}
	if st.field == flowFieldSave {
		btnMark2 = Paint(Accent).Bold(true).Render("▸ ")
	}
	btn1 := btnMark1 + Pill("+ Añadir Fase al Ciclo", "#FFFFFF", "#0C4A6E")
	btn2 := btnMark2 + Pill("✔ Guardar Flujo", "#FFFFFF", "#14532D")
	out = append(out, "  "+btn1+"    "+btn2, "")

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
	offset := 0
	if len(m.flows.phases) > 0 {
		offset = 2
	}
	switch line {
	case 4 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldTemplate}
	case 5 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldName}
	case 6 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPhaseName}
	case 7 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldEngine}
	case 8 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldModel}
	case 9 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldEffort}
	case 10 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldThinking}
	case 11 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldFeedOutput}
	case 12 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldWait}
	case 13 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPrompt}
	case 15 + offset:
		if x < 35 {
			return Target{Kind: TargetFlowItem, Field: "add_phase"}
		}
		return Target{Kind: TargetFlowItem, Field: "save"}
	}
	return Target{}
}
