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

	boxW, innerW := composeBoxWidth(w), composeInnerWidth(w)
	lines := m.composeTextLines(innerW, active,
		Paint(Dim).Render(p.T("compose.text_placeholder", "what is to be done?")))

	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#334155"))
	if active {
		borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#38BDF8"))
	}

	out := []string{header, fit("  "+borderStyle.Render("┌"+strings.Repeat("─", boxW-2)+"┐"), w)}

	for _, l := range lines {
		padW := max(innerW-lipgloss.Width(l), 0)
		row := "  " + borderStyle.Render("│ ") + l + strings.Repeat(" ", padW) + borderStyle.Render(" │")
		out = append(out, fit(row, w))
	}

	out = append(out, fit("  "+borderStyle.Render("└"+strings.Repeat("─", boxW-2)+"┘"), w))

	return out
}

// composeTextLines is what is drawn inside the box: the lines the task
// wraps into, the ones of them that fit, and the caret on the one it is on.
//
// The caret used to be hung off the end of the last drawn line, which after
// the box was padded out to three was an empty row below the text. It goes
// where the reader is instead, and the box scrolls to keep it in view.
func (m Model) composeTextLines(innerW int, active bool, placeholder string) []string {
	rs := m.compose.text.runes()
	spans := wrapSpans(rs, innerW)
	caretRow := spanRow(spans, m.compose.text.at)
	top := spanWindow(len(spans), composeTextRows, caretRow)
	from, to := m.compose.text.selection()

	var out []string

	for i := top; i < len(spans) && i < top+composeTextRows; i++ {
		s := spans[i]

		line := spanText(rs, s)

		if active {
			caret := -1
			if i == caretRow {
				caret = m.compose.text.at - s.from
			}

			line = paintCells(line, from-s.from, to-s.from, caret, unpainted)
		}

		out = append(out, line)
	}

	if len(rs) == 0 {
		out = []string{placeholder}
		if active {
			out = []string{paintCells("", 0, 0, 0, unpainted) + placeholder}
		}
	}

	for len(out) < 3 {
		out = append(out, "")
	}

	return out
}

// unpainted is the box: what is typed into it is drawn as it was typed.
func unpainted(s string) string { return s }

func (m Model) composeFieldLine(fieldIdx int, label string, val input, placeholder string, w int) string {
	active := m.compose.field == fieldIdx
	prefix := composeLabel(label, active)

	if val.empty() {
		line := prefix
		if active {
			line += paintCells("", 0, 0, 0, unpainted)
		}

		return fit(line+Paint(Dim).Render(placeholder), w)
	}

	body := Paint(Accent).Render(val.String())
	if active {
		from, to := val.selection()
		body = paintCells(val.String(), from, to, val.at, func(s string) string { return Paint(Accent).Render(s) })
	}

	return fit(prefix+body, w)
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
