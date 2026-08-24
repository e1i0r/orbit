package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/words"
)

func (m Model) flowsRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	if m.flows.creating {
		return m.flowsBuilderRows(h, w)
	}
	p := m.opts.Words
	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("flows.title", "Flows")),
		"  " + Paint(Dim).Render(p.T("flows.read_only", "flows are read-only; edit flow files under $ORBIT_HOME/flows/ to change them")),
		"",
		"  " + Pill(p.T("flows.create_btn", "+ Create Custom Flow"), "#FFFFFF", "#005F87") + "  " + Paint(Dim).Render("(press n)"),
		"",
	}

	descriptors := flow.List(m.opts.Flows)
	if len(descriptors) == 0 {
		out = append(out, "  "+Paint(Dim).Render(p.T("flows.none", "no flows found")))
	}

	for i, d := range descriptors {
		mark := strings.Repeat(" ", gutter)
		if i == m.flows.sel {
			mark = markGlyph + strings.Repeat(" ", gutter-1)
		}
		originStr := flowOriginString(p, d.Origin)
		headerLine := mark + Paint(Accent).Render(d.Name)
		if originStr != "" {
			headerLine += "  " + Paint(Dim).Render("("+originStr+")")
		}
		out = append(out, fit(headerLine, w))

		fl, err := flow.Resolve(m.opts.Flows, d.Name)
		if err != nil {
			errLine := strings.Repeat(" ", gutter+2) + Paint(Bad).Render(err.Error())
			out = append(out, fit(errLine, w))
			continue
		}

		for idx, ph := range fl.Phases {
			engineModel := ph.Engine
			if ph.Model != "" {
				engineModel += " / " + ph.Model
			}
			feed := ""
			if ph.FeedOutput {
				feed = " [feeds input]"
			}
			waitStr := p.T("flow.runs_auto", "runs automatically")
			if ph.Wait {
				waitStr = p.T("flow.stops_for_human", "stops for human")
			}

			phaseLine := fmt.Sprintf("%s%d. %s  %s%s  (%s)",
				strings.Repeat(" ", gutter+2),
				idx+1,
				Paint(Accent).Render(ph.Name),
				Paint(Dim).Render(engineModel),
				Paint(Live).Render(feed),
				Paint(Dim).Render(waitStr),
			)
			if ph.Prompt != "" {
				phaseLine += "  " + Paint(Dim).Render(`"`+ph.Prompt+`"`)
			}
			out = append(out, fit(phaseLine, w))
		}
		out = append(out, "")
	}

	waysOut := p.T("flows.ways_out", "[n] create · {up_down} scroll · {back} back",
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key),
		about("back", m.keys.Back.Help().Key))
	out = append(out, fit("  "+Paint(Dim).Render(waysOut), w))
	return fill(out, h)
}

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

func flowOriginString(p *words.Printer, o flow.Origin) string {
	switch o {
	case flow.OriginBuiltin:
		return p.T("flow.built_in", "built in")
	case flow.OriginUser:
		return p.T("flow.yours", "yours")
	case flow.OriginShadow:
		return p.T("flow.shadowing", "yours, shadowing the built-in")
	}
	return ""
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
		return Target{}
	}
	offset := 0
	if len(m.flows.phases) > 0 {
		offset = 2
	}
	switch line {
	case 4 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldName}
	case 5 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPhaseName}
	case 6 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldEngine}
	case 7 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldModel}
	case 8 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldEffort}
	case 9 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldThinking}
	case 10 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldFeedOutput}
	case 11 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldWait}
	case 12 + offset:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPrompt}
	case 14 + offset:
		if x < 35 {
			return Target{Kind: TargetFlowItem, Field: "add_phase"}
		}
		return Target{Kind: TargetFlowItem, Field: "save"}
	}
	return Target{}
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
