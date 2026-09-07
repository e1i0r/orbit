package compose

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/ui/typing"
)

// View draws the form: the tabs, the flow and what it will do, the fields a
// task is written into, the issue preview, and the actions at the foot.
func (s State) View(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	p := e.Words

	var out []string

	tabManual := p.T("compose.tab_manual", "1 Manual")
	tabURL := p.T("compose.tab_url", "2 From URL (Linear / Jira / GitHub)")

	renderTab := func(name string, active bool) string {
		if active {
			return theme.Paint(theme.Sel).Bold(true).Render(" [ " + name + " ] ")
		}

		return theme.Paint(theme.Dim).Render("   " + name + "   ")
	}

	tabLine := "  " + renderTab(tabManual, s.tab == composeTabManual) +
		" " + renderTab(tabURL, s.tab == composeTabURL)
	out = append(out, tabLine, "")

	if s.tab == composeTabManual {
		out = append(out, s.composeManualRows(w, e)...)
	} else {
		out = append(out, s.composeURLRows(w, e)...)
	}

	saveBtn := p.T("compose.save_btn", "↵ Save")
	runBtn := p.T("compose.save_run_btn", "^R Save & Run")
	cancelBtn := p.T("compose.cancel_btn", "esc Cancel")

	actions := "  " + theme.Paint(theme.OK).Render("[ "+saveBtn+" ]") +
		"   " + theme.Paint(theme.Accent).Render("[ "+runBtn+" ]") +
		"   " + theme.Paint(theme.Dim).Render("[ "+cancelBtn+" ]")

	if e.Autopilot {
		actions += "   " + theme.Paint(theme.Accent).Render("⚡ "+p.T("compose.autopilot_on_note", "autopilot is ON: starts automatically [A to toggle]"))
	} else {
		actions += "   " + theme.Paint(theme.Dim).Render("⚡ "+p.T("compose.autopilot_off_note", "autopilot is OFF: saves to To Do backlog"))
	}

	out = append(out, "", cells.Fit(actions, w))

	return cells.Fill(out, h)
}

// composeFlowDetail is the block under the flow pills: the phases the chosen
// flow will run, and what it says about itself. The first row carries the ↳
// and the rest line up under it, so the block reads as one answer to the
// field above rather than as loose lines in the form.
func (s State) composeFlowDetail(w int, e Env) []string {
	rows := s.flowDetail(s.chosenFlow(), w, e)
	padded := strings.Repeat(" ", cells.Gutter+composeLabelWidth+1)

	out := make([]string, 0, len(rows))

	for i, row := range rows {
		lead := "  "
		if i == 0 {
			lead = "↳ "
		}

		out = append(out, cells.Fit(padded+theme.Paint(theme.Dim).Render(lead)+theme.Paint(theme.Dim).Render(row), w))
	}

	return out
}

func (s State) composeManualRows(w int, e Env) []string {
	p := e.Words

	var out []string

	out = append(out, s.composeFlowLine(s.field == composeFlow, w, e))
	out = append(out, s.composeFlowDetail(w, e)...)

	idLine := s.composeFieldLine(
		composeID,
		p.T("compose.id", "id"),
		s.id,
		p.T("compose.id_placeholder", "what is it called? (e.g. ORBIT-42)"),
		w,
	)
	out = append(out, idLine)
	out = append(out, s.composeBox(
		composeText,
		p.T("compose.text", "task"),
		p.T("compose.text_placeholder", "what is to be done?"),
		p.T("compose.text_hint", "(Shift+↵ for newline)"),
		s.text,
		w,
		e,
	)...)

	return out
}

func (s State) composeURLRows(w int, e Env) []string {
	var out []string

	out = append(out, s.composeFlowLine(s.field == composeURLFlow, w, e))
	out = append(out, s.composeFlowDetail(w, e)...)

	if s.parsedIssue != nil {
		iss := s.parsedIssue

		preview := "  " + theme.Paint(theme.OK).Render("✓ "+strings.ToUpper(iss.Kind)) +
			" · " + theme.Paint(theme.Accent).Render(iss.ID)
		if iss.Title != "" {
			preview += " · " + theme.Paint(theme.Dim).Render(iss.Title)
		}

		out = append(out, "", cells.Fit(preview, w))
	}
	// Last, under the flow and under what that flow says it will do. The
	// URL is the one thing this tab is for, so it is the row the eye ends
	// on and the row the actions are typed from.
	//
	// It is a box for the same reason the task is one: a tracker URL is
	// long, and a row of the form cuts it off at the width of the window.
	// A reader who cannot see the end of what they pasted cannot tell a
	// URL that is wrong from one that is merely far away.
	p := e.Words

	return append(out, s.composeBox(
		composeURL,
		p.T("compose.url", "url"),
		p.T("compose.url_placeholder", "https://linear.app/... or https://...atlassian.net/..."),
		"",
		s.url,
		w,
		e,
	)...)
}

func (s State) composeFieldLine(fieldIdx int, label string, val typing.Field, placeholder string, w int) string {
	active := s.field == fieldIdx
	prefix := composeLabel(label, active)

	if val.Empty() {
		line := prefix
		if active {
			line += typing.PaintCells("", 0, 0, 0, unpainted)
		}

		return cells.Fit(line+theme.Paint(theme.Dim).Render(placeholder), w)
	}

	body := theme.Paint(theme.Accent).Render(val.String())
	if active {
		from, to := val.Selection()
		body = typing.PaintCells(val.String(), from, to, val.At, func(s string) string { return theme.Paint(theme.Accent).Render(s) })
	}

	return cells.Fit(prefix+body, w)
}
