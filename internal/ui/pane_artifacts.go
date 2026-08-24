package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

type runFile struct {
	name        string
	size        string
	description string
}

func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", max(bytes, 1))
	}
	return fmt.Sprintf("%d k", bytes/1024)
}

// artifactsLines renders Pane 8: Files and artifacts left by the task & run.
func (m Model) artifactsLines() []string {
	p := m.opts.Words
	t, ok := m.task(m.detail)
	if !ok {
		return []string{"  " + Paint(Dim).Render(p.T("detail.gone", "this task is no longer on the board"))}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("artifacts.title", "Files & Artifacts")),
		"  " + Paint(Dim).Render(p.T("artifacts.subtitle", "every file left by the run, and what each one is")),
		"",
	}

	// 1. Lo que produjo (Produced by model/engine)
	var produced []runFile
	if m.diffKnown && m.diff != "" {
		sum := parseDiffSummary(m.diff)
		for _, f := range sum.files {
			produced = append(produced, runFile{
				name:        f,
				size:        "modified",
				description: p.T("artifacts.desc_new_file", "new file created during the run"),
			})
		}
	}
	// Check if report or model outputs exist in task events
	for _, e := range m.entries {
		if e.Phase != "" && e.Text != "" {
			produced = append(produced, runFile{
				name:        fmt.Sprintf("report-%s.md", strings.ToLower(e.Phase)),
				size:        formatBytes(int64(len(e.Text))),
				description: p.T("artifacts.desc_report", "draft summary report in model's own words"),
			})
		}
	}

	if len(produced) > 0 {
		out = append(out, "  "+Paint(Accent).Render(p.T("artifacts.group_produced", "what it produced")))
		for _, rf := range produced {
			out = append(out, fmt.Sprintf("    %-28s  %-8s  %s",
				Paint(Accent).Render(rf.name),
				Paint(Dim).Render(rf.size),
				Paint(Dim).Render(rf.description),
			))
		}
		out = append(out, "")
	}

	// 2. Los chequeos (Gates / Verifications)
	var checks []runFile
	for _, e := range m.entries {
		if e.Gate != "" || e.What() == view.EntryWaiting {
			checks = append(checks, runFile{
				name:        "gates.json",
				size:        "380 B",
				description: p.T("artifacts.desc_gate_json", "each check, its verdict and how long it took"),
			})
			if e.Cause != "" {
				checks = append(checks, runFile{
					name:        "gates.reason",
					size:        formatBytes(int64(len(e.Cause))),
					description: p.T("artifacts.desc_gate_reason", "one sentence: why the gate stopped"),
				})
			}
			break
		}
	}
	if len(checks) > 0 {
		out = append(out, "  "+Paint(Accent).Render(p.T("artifacts.group_checks", "the checks & gates")))
		for _, rf := range checks {
			out = append(out, fmt.Sprintf("    %-28s  %-8s  %s",
				Paint(Accent).Render(rf.name),
				Paint(Dim).Render(rf.size),
				Paint(Dim).Render(rf.description),
			))
		}
		out = append(out, "")
	}

	// 3. Lo que se le pidió (Input / Prompts)
	var inputs []runFile
	inputs = append(inputs, runFile{
		name:        "task.md",
		size:        formatBytes(int64(len(t.Title))),
		description: p.T("artifacts.desc_task_md", "the task prompt, exactly as written"),
	})
	if t.Flow != "" {
		inputs = append(inputs, runFile{
			name:        "task.env",
			size:        "128 B",
			description: p.T("artifacts.desc_task_env", "environment fields and configuration read by runner"),
		})
	}
	out = append(out, "  "+Paint(Accent).Render(p.T("artifacts.group_prompts", "what it was asked for")))
	for _, rf := range inputs {
		out = append(out, fmt.Sprintf("    %-28s  %-8s  %s",
			Paint(Accent).Render(rf.name),
			Paint(Dim).Render(rf.size),
			Paint(Dim).Render(rf.description),
		))
	}
	out = append(out, "")

	// 4. La contabilidad del run (Accounting & logs)
	var accounting []runFile
	eventsSize := int64(len(m.entries) * 120)
	if eventsSize == 0 {
		eventsSize = 120
	}
	accounting = append(accounting, runFile{
		name:        "events.jsonl",
		size:        formatBytes(eventsSize),
		description: p.T("artifacts.desc_events", "the immutable event log for this task"),
	})
	if t.Cost > 0 {
		accounting = append(accounting, runFile{
			name:        "cost.tsv",
			size:        "45 B",
			description: p.T("artifacts.desc_cost", "one row per phase: what it cost in $"),
		})
	}
	accounting = append(accounting, runFile{
		name:        "state",
		size:        "8 B",
		description: p.T("artifacts.desc_state", "the phase it was in when last written"),
	})
	out = append(out, "  "+Paint(Accent).Render(p.T("artifacts.group_accounting", "run accounting")))
	for _, rf := range accounting {
		out = append(out, fmt.Sprintf("    %-28s  %-8s  %s",
			Paint(Accent).Render(rf.name),
			Paint(Dim).Render(rf.size),
			Paint(Dim).Render(rf.description),
		))
	}
	out = append(out, "")

	return out
}
