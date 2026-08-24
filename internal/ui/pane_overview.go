package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
)

// overviewLines renders Pane 1: Overview of the task.
func (m Model) overviewLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.logErr.Error())}
	}
	t, ok := m.task(m.detail)
	if !ok {
		return []string{"  " + Paint(Dim).Render(p.T("detail.gone", "this task is no longer on the board"))}
	}

	word, role := m.stateWord(t)
	var out []string

	// Title / Goal header
	out = append(out,
		"",
		"  "+Paint(Accent).Render(t.ID)+" "+Paint(Dim).Render("·")+" "+Paint(role).Render(word),
		"  "+Paint(Dim).Render(p.T("overview.goal", "qué es esta tarea, y hasta dónde llegó")),
	)
	if t.Title != "" {
		out = append(out, "", "  "+t.Title)
	}
	out = append(out, "")

	// 1. Qué pasó (What happened)
	out = append(out,
		"  "+Paint(Accent).Render("qué pasó"),
		fmt.Sprintf("    %-12s %s", Paint(Dim).Render("duró"), elapsed(m.now, t.Since)),
		fmt.Sprintf("    %-12s %s", Paint(Dim).Render("terminó"), Paint(role).Render(word)),
	)
	if t.Cost > 0 {
		out = append(out, fmt.Sprintf("    %-12s %s", Paint(Dim).Render("gastó"), fmt.Sprintf("$%.4f", t.Cost)))
	}
	out = append(out, "")

	// 2. Dónde está (Where it is)
	out = append(out,
		"  "+Paint(Accent).Render("dónde está"),
		fmt.Sprintf("    %-12s %s", Paint(Dim).Render("repo"), t.Repo),
	)
	if t.RepoPath != "" {
		out = append(out, fmt.Sprintf("    %-12s %s", Paint(Dim).Render("worktree"), t.RepoPath))
	}
	if t.Flow != "" {
		out = append(out, fmt.Sprintf("    %-12s %s", Paint(Dim).Render("flujo"), t.Flow))
	}
	out = append(out, "")

	// 3. Waiting / Attention Block
	if role == Warn || role == Bad {
		out = append(out,
			"  "+Pill("⚠️ "+p.T("overview.waiting_box", "IT IS WAITING ON YOU / ACCIONES REQUERIDAS"), "#000000", "#FBBF24"),
			"  "+Paint(Warn).Render("status: "+word),
			"  "+Paint(Dim).Render(p.T("overview.resume_hint", "press 't' to open interactive session, 'r' to restart")),
			"",
		)
	}

	return out
}

// flowLines renders Pane 2: Flow & Phase Breakdown.
func (m Model) flowLines() []string {
	p := m.opts.Words
	t, ok := m.task(m.detail)
	if !ok {
		return []string{"  " + Paint(Dim).Render(p.T("detail.gone", "this task is no longer on the board"))}
	}

	flowName := t.Flow
	if flowName == "" {
		flowName = "quick"
	}
	f, err := flow.Resolve(m.opts.Flows, flowName)
	if err != nil {
		return []string{"  " + Paint(Bad).Render(fmt.Sprintf("flow %q: %v", flowName, err))}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("flow.title", "Flow: {name}", about("name", f.Name))),
		"",
	}

	for i, phase := range f.Phases {
		tag := fmt.Sprintf("[%d/%d] %s", i+1, len(f.Phases), phase.Name)
		var details []string
		if phase.Engine != "" {
			details = append(details, phase.Engine)
		}
		if phase.Model != "" {
			details = append(details, phase.Model)
		}
		if phase.Effort != "" {
			details = append(details, "effort:"+phase.Effort)
		}
		if phase.Thinking != "" {
			details = append(details, "thinking:"+phase.Thinking)
		}
		if len(phase.Permissions) > 0 {
			details = append(details, "perms:"+strings.Join(phase.Permissions, ","))
		}
		if phase.Wait {
			details = append(details, "stops at gate")
		}
		if len(phase.Gates) > 0 {
			details = append(details, fmt.Sprintf("%d verification gate(s)", len(phase.Gates)))
		}

		out = append(out,
			"  "+Paint(Accent).Render(tag),
			"      "+Paint(Dim).Render(strings.Join(details, " · ")),
		)
		for _, g := range phase.Gates {
			out = append(out, "      "+Paint(Dim).Render("gate "+g.Name+": ")+g.Command)
		}
		out = append(out, "")
	}

	return out
}
