package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// composeRows draws the compose screen: top tabs, fields, repository selector,
// issue preview, and action buttons.
func (m Model) composeRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	p := m.opts.Words
	var out []string

	tabManual := p.T("compose.tab_manual", "1 Manual")
	tabURL := p.T("compose.tab_url", "2 Desde URL (Linear / Jira / GitHub)")

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

	saveBtn := p.T("compose.save_btn", "↵ Guardar")
	runBtn := p.T("compose.save_run_btn", "^R Guardar y Ejecutar")
	cancelBtn := p.T("compose.cancel_btn", "esc Cancelar")

	actions := "  " + Paint(OK).Render("[ "+saveBtn+" ]") +
		"   " + Paint(Accent).Render("[ "+runBtn+" ]") +
		"   " + Paint(Dim).Render("[ "+cancelBtn+" ]")

	out = append(out, "", fit(actions, w))
	return fill(out, h)
}

func (m Model) composeManualRows(w int) []string {
	p := m.opts.Words
	var out []string

	repoLine := m.composeRepoLine(m.compose.field == composeRepo, w)
	out = append(out, repoLine)

	idLine := m.composeFieldLine(
		composeID,
		p.T("compose.id", "id"),
		m.compose.id,
		p.T("compose.id_placeholder", "what is it called? (e.g. ORBIT-42)"),
		w,
	)
	out = append(out, idLine)

	textLine := m.composeFieldLine(
		composeText,
		p.T("compose.text", "task"),
		m.compose.text,
		p.T("compose.text_placeholder", "what is to be done?"),
		w,
	)
	out = append(out, textLine)
	return out
}

func (m Model) composeURLRows(w int) []string {
	p := m.opts.Words
	var out []string

	urlLine := m.composeFieldLine(
		composeURL,
		p.T("compose.url", "url"),
		m.compose.url,
		p.T("compose.url_placeholder", "https://linear.app/... o https://...atlassian.net/..."),
		w,
	)
	out = append(out, urlLine)

	repoLine := m.composeRepoLine(m.compose.field == composeURLRepo, w)
	out = append(out, repoLine)

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

func (m Model) composeRepoLine(active bool, w int) string {
	p := m.opts.Words
	mark := strings.Repeat(" ", gutter)
	if active {
		mark = markGlyph + strings.Repeat(" ", gutter-1)
	}
	prefix := mark + Paint(Dim).Render(p.T("compose.repo", "repository")+": ")

	if len(m.compose.repos) == 0 {
		val := m.compose.repo
		role := Accent
		if val == "" {
			val = p.T("compose.repo_placeholder", "which repository?")
			role = Dim
		}
		line := prefix + Paint(role).Render(val)
		if active {
			line += Paint(Sel).Render(" ")
		}
		return fit(line, w)
	}

	var pills []string
	for i, r := range m.compose.repos {
		selected := i == m.compose.repoIdx
		if selected {
			pills = append(pills, Pill(" ● "+r.name+" ", "#000000", "#38BDF8"))
		} else {
			pills = append(pills, Pill(" "+r.name+" ", "#94A3B8", "#1E293B"))
		}
	}
	line := prefix + strings.Join(pills, " ")
	if active {
		line += " " + Paint(Dim).Render(p.T("compose.repo_hint", "(←/→ para cambiar)"))
	}
	return fit(line, w)
}

func (m Model) composeFieldLine(fieldIdx int, label, val, placeholder string, w int) string {
	active := m.compose.field == fieldIdx
	mark := strings.Repeat(" ", gutter)
	if active {
		mark = markGlyph + strings.Repeat(" ", gutter-1)
	}
	role := Accent
	if val == "" {
		val = placeholder
		role = Dim
	}
	line := mark + Paint(Dim).Render(label+": ") + Paint(role).Render(val)
	if active {
		line += Paint(Sel).Render(" ")
	}
	return fit(line, w)
}

// composeRepoPillLen returns the rendered width of a repo pill.
func composeRepoPillLen(name string, selected bool) int {
	if selected {
		return lipgloss.Width(" ● "+name+" ") + 1
	}
	return lipgloss.Width(" "+name+" ") + 1
}
