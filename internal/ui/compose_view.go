package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// composeRows draws the compose screen: top tabs, the flow and what it will
// do, the fields a task is written into, the issue preview, and the actions.
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

// composeFlowDetail is the block under the flow pills: the phases the chosen
// flow will run, and what it says about itself. The first row carries the ↳
// and the rest line up under it, so the block reads as one answer to the
// field above rather than as loose lines in the form.
func (m Model) composeFlowDetail(w int) []string {
	rows := m.flowDetail(m.compose.chosenFlow(), w)
	padded := strings.Repeat(" ", gutter+composeLabelWidth+1)

	out := make([]string, 0, len(rows))

	for i, row := range rows {
		lead := "  "
		if i == 0 {
			lead = "↳ "
		}

		out = append(out, fit(padded+Paint(Dim).Render(lead)+Paint(Dim).Render(row), w))
	}

	return out
}

func (m Model) composeManualRows(w int) []string {
	p := m.opts.Words

	var out []string

	out = append(out, m.composeFlowLine(m.compose.field == composeFlow, w))
	out = append(out, m.composeFlowDetail(w)...)

	idLine := m.composeFieldLine(
		composeID,
		p.T("compose.id", "id"),
		m.compose.id,
		p.T("compose.id_placeholder", "what is it called? (e.g. ORBIT-42)"),
		w,
	)
	out = append(out, idLine)
	out = append(out, m.composeBox(
		composeText,
		p.T("compose.text", "task"),
		p.T("compose.text_placeholder", "what is to be done?"),
		p.T("compose.text_hint", "(Shift+↵ for newline)"),
		m.compose.text,
		w,
	)...)

	return out
}

func (m Model) composeURLRows(w int) []string {
	var out []string

	out = append(out, m.composeFlowLine(m.compose.field == composeURLFlow, w))
	out = append(out, m.composeFlowDetail(w)...)

	if m.compose.parsedIssue != nil {
		iss := m.compose.parsedIssue

		preview := "  " + Paint(OK).Render("✓ "+strings.ToUpper(iss.Kind)) +
			" · " + Paint(Accent).Render(iss.ID)
		if iss.Title != "" {
			preview += " · " + Paint(Dim).Render(iss.Title)
		}

		out = append(out, "", fit(preview, w))
	}
	// Last, under the flow and under what that flow says it will do. The
	// URL is the one thing this tab is for, so it is the row the eye ends
	// on and the row the actions are typed from.
	//
	// It is a box for the same reason the task is one: a tracker URL is
	// long, and a row of the form cuts it off at the width of the window.
	// A reader who cannot see the end of what they pasted cannot tell a
	// URL that is wrong from one that is merely far away.
	p := m.opts.Words

	return append(out, m.composeBox(
		composeURL,
		p.T("compose.url", "url"),
		p.T("compose.url_placeholder", "https://linear.app/... or https://...atlassian.net/..."),
		"",
		m.compose.url,
		w,
	)...)
}

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
