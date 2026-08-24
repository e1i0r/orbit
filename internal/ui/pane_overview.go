package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
)

// overviewLines renders Pane 1: Overview of the task.
func (m Model) overviewLines() []string {
	p := m.opts.Words
	t, ok := m.task(m.detail)
	if !ok {
		return []string{"  " + Paint(Dim).Render(p.T("detail.gone", "this task is no longer on the board"))}
	}

	word, role := m.stateWord(t)
	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("overview.heading", "Task Overview")),
		"",
		fmt.Sprintf("    %-16s %s", Paint(Dim).Render(p.T("overview.id", "ID:")), Paint(Accent).Render(t.ID)),
		fmt.Sprintf("    %-16s %s", Paint(Dim).Render(p.T("overview.repo", "Repository:")), t.Repo+" ("+t.RepoPath+")"),
		fmt.Sprintf("    %-16s %s", Paint(Dim).Render(p.T("overview.status", "Status:")), Paint(role).Render(word)),
	}
	if t.Flow != "" {
		out = append(out, fmt.Sprintf("    %-16s %s", Paint(Dim).Render(p.T("overview.flow", "Flow:")), t.Flow))
	}
	if t.Cost > 0 {
		out = append(out, fmt.Sprintf("    %-16s %s", Paint(Dim).Render(p.T("overview.cost", "Cost:")),
			fmt.Sprintf("$%.4f", t.Cost)))
	}
	if t.Title != "" {
		out = append(out, fmt.Sprintf("    %-16s %s", Paint(Dim).Render(p.T("overview.title", "Title:")), t.Title))
	}

	var taskText string
	for _, e := range m.entries {
		if e.What() == view.EntryWritten {
			taskText = e.Text
			break
		}
	}
	if taskText != "" {
		out = append(out,
			"",
			"  "+Paint(Accent).Render(p.T("overview.description", "Description:")),
			"",
		)
		for _, line := range strings.Split(taskText, "\n") {
			out = append(out, "    "+line)
		}
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
