package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// composeRows draws the compose screen: top tabs, fields, repository selector,
// flow, engine, model, thinking, effort selectors, issue preview, and action buttons.
func (m Model) composeRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	p := m.opts.Words
	var out []string

	tabManual := p.T("compose.tab_manual", "1 Manual")
	tabURL := p.T("compose.tab_url", "2 From URL (Linear / Jira / GitHub)")

	renderTab := func(name string, active bool) string {
		if active {
			return Paint(Sel).Bold(true).Render(" [ " + name + " ] ")
		}
		return Paint(Dim).Render("   " + name + "   ")
	}

	tabLine := "  " + renderTab(tabManual, m.compose.tab == composeTabManual) +
		" " + renderTab(tabURL, m.compose.tab == composeTabURL)
	out = append(out, tabLine, "")

	if m.compose.tab == composeTabManual {
		out = append(out, m.composeManualRows(w)...)
	} else {
		out = append(out, m.composeURLRows(w)...)
	}

	saveBtn := p.T("compose.save_btn", "↵ Save")
	runBtn := p.T("compose.save_run_btn", "^R Save & Run")
	cancelBtn := p.T("compose.cancel_btn", "esc Cancel")

	actions := "  " + Paint(OK).Render("[ "+saveBtn+" ]") +
		"   " + Paint(Accent).Render("[ "+runBtn+" ]") +
		"   " + Paint(Dim).Render("[ "+cancelBtn+" ]")

	if m.autopilotOn() {
		actions += "   " + Paint(Accent).Render("⚡ "+p.T("compose.autopilot_on_note", "autopilot is ON: starts automatically [A to toggle]"))
	} else {
		actions += "   " + Paint(Dim).Render("⚡ "+p.T("compose.autopilot_off_note", "autopilot is OFF: saves to To Do backlog"))
	}

	out = append(out, "", fit(actions, w))
	return fill(out, h)
}

func (m Model) composeManualRows(w int) []string {
	p := m.opts.Words
	var out []string

	out = append(out, m.composeRepoLine(m.compose.field == composeRepo, w))
	out = append(out, m.composeFlowLine(m.compose.field == composeFlow, w))
	if sum := m.flowSummary(m.compose.chosenFlow()); sum != "" {
		padded := strings.Repeat(" ", gutter+composeLabelWidth+1)
		out = append(out, fit(padded+Paint(Dim).Render("↳ "+sum), w))
	}
	out = append(out, m.composeEngineLine(m.compose.field == composeEngine, w))
	out = append(out, m.composeModelLine(m.compose.field == composeModel, w))
	out = append(out, m.composeThinkingLine(m.compose.field == composeThinking, w))
	out = append(out, m.composeEffortLine(m.compose.field == composeEffort, w))

	idLine := m.composeFieldLine(
		composeID,
		p.T("compose.id", "id"),
		m.compose.id,
		p.T("compose.id_placeholder", "what is it called? (e.g. ORBIT-42)"),
		w,
	)
	out = append(out, idLine)
	out = append(out, m.composeTextArea(w)...)
	return out
}

func (m Model) composeURLRows(w int) []string {
	p := m.opts.Words
	var out []string

	urlLine := m.composeFieldLine(
		composeURL,
		p.T("compose.url", "url"),
		m.compose.url,
		p.T("compose.url_placeholder", "https://linear.app/... or https://...atlassian.net/..."),
		w,
	)
	pastePill := Pill(" 📋 "+p.T("compose.btn_paste", "Paste (^V)")+" ", "#FFFFFF", "#0369A1")
	urlLine += " " + pastePill
	out = append(out, urlLine)

	out = append(out, m.composeRepoLine(m.compose.field == composeURLRepo, w))
	out = append(out, m.composeFlowLine(m.compose.field == composeURLFlow, w))
	if sum := m.flowSummary(m.compose.chosenFlow()); sum != "" {
		padded := strings.Repeat(" ", gutter+composeLabelWidth+1)
		out = append(out, fit(padded+Paint(Dim).Render("↳ "+sum), w))
	}
	out = append(out, m.composeEngineLine(m.compose.field == composeURLEngine, w))
	out = append(out, m.composeModelLine(m.compose.field == composeURLModel, w))
	out = append(out, m.composeThinkingLine(m.compose.field == composeURLThinking, w))
	out = append(out, m.composeEffortLine(m.compose.field == composeURLEffort, w))

	if m.compose.parsedIssue != nil {
		iss := m.compose.parsedIssue
		preview := "  " + Paint(OK).Render("✓ "+strings.ToUpper(iss.Kind)) +
			" · " + Paint(Accent).Render(iss.ID)
		if iss.Title != "" {
			preview += " · " + Paint(Dim).Render(iss.Title)
		}
		out = append(out, "", fit(preview, w))
	}
	return out
}

func (m Model) composeTextArea(w int) []string {
	p := m.opts.Words
	active := m.compose.field == composeText
	header := composeLabel(p.T("compose.text", "task"), active)
	pastePill := Pill(" 📋 "+p.T("compose.btn_paste", "Paste (^V)")+" ", "#FFFFFF", "#0369A1")
	header += pastePill
	if active {
		header += " " + Paint(Dim).Render(p.T("compose.text_hint", "(Shift+↵ for newline)"))
	}

	boxW := w - 8
	if boxW < 24 {
		boxW = 24
	}
	if boxW > 84 {
		boxW = 84
	}
	innerW := boxW - 4

	raw := m.compose.text
	var lines []string
	if raw == "" {
		lines = []string{Paint(Dim).Render(p.T("compose.text_placeholder", "what is to be done?"))}
	} else {
		for _, part := range strings.Split(raw, "\n") {
			if part == "" {
				lines = append(lines, "")
			} else {
				lines = append(lines, splitIntoLines(part, innerW)...)
			}
		}
	}
	for len(lines) < 3 {
		lines = append(lines, "")
	}
	if len(lines) > 6 {
		lines = lines[len(lines)-6:]
	}

	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#334155"))
	if active {
		borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#38BDF8"))
	}

	var out []string
	out = append(out, header)
	out = append(out, fit("  "+borderStyle.Render("┌"+strings.Repeat("─", boxW-2)+"┐"), w))
	for i, l := range lines {
		content := l
		if active && i == len(lines)-1 && raw != "" {
			content += Paint(Sel).Render(" ")
		}
		wLine := lipgloss.Width(content)
		padW := innerW - wLine
		if padW < 0 {
			padW = 0
		}
		row := "  " + borderStyle.Render("│ ") + content + strings.Repeat(" ", padW) + borderStyle.Render(" │")
		out = append(out, fit(row, w))
	}
	out = append(out, fit("  "+borderStyle.Render("└"+strings.Repeat("─", boxW-2)+"┘"), w))
	return out
}

func (m Model) composeFieldLine(fieldIdx int, label, val, placeholder string, w int) string {
	active := m.compose.field == fieldIdx
	prefix := composeLabel(label, active)
	role := Accent
	if val == "" {
		val = placeholder
		role = Dim
	}
	line := prefix + Paint(role).Render(val)
	if active {
		line += Paint(Sel).Render(" ")
	}
	return fit(line, w)
}

func splitIntoLines(text string, maxW int) []string {
	if maxW <= 0 {
		return []string{text}
	}
	var res []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var cur strings.Builder
	for _, w := range words {
		switch {
		case cur.Len() == 0:
			cur.WriteString(w)
		case lipgloss.Width(cur.String())+1+lipgloss.Width(w) <= maxW:
			cur.WriteString(" " + w)
		default:
			res = append(res, cur.String())
			cur.Reset()
			cur.WriteString(w)
		}
	}
	if cur.Len() > 0 {
		res = append(res, cur.String())
	}
	return res
}
