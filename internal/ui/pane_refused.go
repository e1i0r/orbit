package ui

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/view"
)

// refusedLines renders Pane 5: Permission denials and refused actions.
func (m Model) refusedLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.logErr.Error())}
	}

	var denials []view.Entry

	for _, e := range m.entries {
		if e.What() == view.EntryRefused {
			denials = append(denials, e)
		}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("refused.title", "Permissions & Security Sandbox")),
		"  " + Paint(Dim).Render(p.T("refused.subtitle", "what the sandbox forbids, and what it attempted")),
		"",
	}

	// 1. En esta corrida
	out = append(out, "  "+Paint(Accent).Render(p.T("refused.in_this_run", "IN THIS RUN")))
	if len(denials) == 0 {
		out = append(out,
			"    "+Paint(OK).Render(p.T("refused.none_denied", "no commands or actions were denied")),
			"    "+Paint(Dim).Render(p.T("refused.all_allowed", "everything it attempted was permitted to run")),
		)
	} else {
		for _, d := range denials {
			toolName := d.Tool
			if toolName == "" {
				toolName = "command"
			}

			out = append(out, fmt.Sprintf("    %s %s: %s",
				Paint(Bad).Render("✗"),
				Paint(Accent).Render(toolName),
				Paint(Bad).Render(d.Text),
			))
		}
	}

	out = append(out, "")

	// 2. Las reglas fijas del sandbox
	out = append(out,
		"  "+Paint(Accent).Render(p.T("refused.rules_title", "THE RULES · sandbox constraints")),
		fmt.Sprintf("    %s %-24s %s", Paint(Dim).Render("✗"), "psql / mongosh", Paint(Dim).Render("bases de datos protegidas (solo lectura)")),
		fmt.Sprintf("    %s %-24s %s", Paint(Dim).Render("✗"), "aws / cloud-cli", Paint(Dim).Render("servicios cloud y credenciales externas")),
		fmt.Sprintf("    %s %-24s %s", Paint(Dim).Render("✗"), "git push", Paint(Dim).Render("la rama la gestiona el operador / runner")),
		fmt.Sprintf("    %s %-24s %s", Paint(Dim).Render("✗"), "git remote / config", Paint(Dim).Render("configuración del repositorio")),
		fmt.Sprintf("    %s %-24s %s", Paint(Dim).Render("✗"), "gh pr merge", Paint(Dim).Render("mezclar y publicar pull requests")),
		"",
		"  "+Paint(Dim).Render(p.T("refused.policy_note", "si intenta una acción prohibida, la llamada falla de inmediato y el modelo continúa")),
		"",
	)

	return out
}
