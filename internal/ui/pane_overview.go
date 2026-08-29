package ui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
)

// diffSummary holds counts of additions, deletions, and modified file paths.
type diffSummary struct {
	added   int
	deleted int
	files   []string
}

func parseDiffSummary(diff string) diffSummary {
	if diff == "" {
		return diffSummary{}
	}

	var sum diffSummary

	seenFiles := make(map[string]bool)

	lines := strings.Split(diff, "\n")
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, "+++ b/"):
			f := strings.TrimPrefix(l, "+++ b/")
			if f != "" && f != "/dev/null" && !seenFiles[f] {
				seenFiles[f] = true
				sum.files = append(sum.files, f)
			}
		case strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++"):
			sum.added++
		case strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---"):
			sum.deleted++
		}
	}

	return sum
}

// overviewLines renders Pane 1: Overview of the task.
func (m Model) overviewLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.errSaid(m.logErr))}
	}

	t, ok := m.task(m.detail)
	if !ok {
		return []string{"  " + Paint(Dim).Render(p.T("detail.gone", "this task is no longer on the board"))}
	}

	word, role := m.stateWord(t)

	var out []string

	// Header: Task ID, status badge and title
	statusBadge := Paint(role).Bold(true).Render("[" + word + "]")

	out = append(out,
		"",
		"  "+Paint(Accent).Bold(true).Render(t.ID)+"  "+statusBadge,
	)
	if t.Title != "" {
		if m.expandedDetail {
			for _, tl := range splitIntoLines(t.Title, max(20, m.frame.Body.W-6)) {
				out = append(out, "  "+Paint(Accent).Render(tl))
			}
		} else {
			out = append(out, "  "+Paint(Accent).Render(t.Title))
		}
	}

	out = append(out, "")

	// 1. Attention/Gate banner (if waiting on user or failed)
	if role == Warn || role == Bad {
		bannerText := p.T("overview.waiting_box", "ACTIONS REQUIRED / WAITING FOR THE OPERATOR")
		out = append(out,
			"  "+Pill(" ⚠️  "+bannerText+" ", "#000000", "#FBBF24"),
			"  "+Paint(role).Render("• "+word),
			"  "+Paint(Dim).Render(p.T("overview.resume_hint", "press 't' to open interactive session, 'r' to restart")),
			"",
		)
	}

	// 2. Metrics & Status Box
	out = append(out,
		"  "+Paint(Accent).Bold(true).Render(p.T("overview.metrics", "Status & Metrics")),
		fmt.Sprintf("    %-14s %s", Paint(Dim).Render(p.T("overview.state", "state")), Paint(role).Render(word)),
		fmt.Sprintf("    %-14s %s", Paint(Dim).Render(p.T("overview.duration", "duration")), elapsed(m.now, t.Since)),
	)
	if t.Cost > 0 {
		out = append(out, fmt.Sprintf("    %-14s %s", Paint(Dim).Render(p.T("overview.cost", "cost")), fmt.Sprintf("$%.4f", t.Cost)))
	}
	// Engine, Model, Effort, Thinking & Flow live indicators
	// A task that has run carries its own engine and model. One that has
	// not shows what it would run on, which is the knob and then the
	// setting behind it — not the words claude and sonnet, which were the
	// answer here on builds that have neither.
	eng := orDef(t.Engine, m.dialEngine(m.knobs.Engine))

	models, _ := m.modelsFor(eng)
	mod := orDef(t.Model, orDef(m.knobs.Model, first(models)))

	// A window whose engines port answers nothing has no engine and no
	// model to name here, and a dash says so without naming one it has not.
	eng, mod = orDef(eng, unsetDial), orDef(mod, unsetDial)

	eff := m.knobs.Effort
	if eff == "" {
		eff = "high"
	}

	thk := m.knobs.Thinking
	if thk == "" {
		thk = "adaptive"
	}

	flowName := t.Flow
	if flowName == "" {
		flowName = flow.Default
	}

	out = append(out,
		fmt.Sprintf("    %-14s %s", Paint(Dim).Render(p.T("overview.engine", "engine")),
			Paint(Live).Render(eng)+" · "+Paint(Accent).Render(mod)+" "+Paint(Dim).Render("[k]")),
		fmt.Sprintf("    %-14s %s", Paint(Dim).Render(p.T("overview.thinking", "thinking")),
			Paint(OK).Render(thk)+" "+Paint(Dim).Render("[t]")),
		fmt.Sprintf("    %-14s %s", Paint(Dim).Render(p.T("overview.effort", "effort")),
			Paint(Accent).Render(eff)+" "+Paint(Dim).Render("[E]")),
		fmt.Sprintf("    %-14s %s", Paint(Dim).Render(p.T("overview.flow", "flow")),
			Paint(Accent).Render(flowName)+" "+Paint(Dim).Render("[F]")),
		fmt.Sprintf("    %-14s %s", Paint(Dim).Render(p.T("overview.repo", "repository")),
			t.Repo+" ("+t.RepoPath+")"),
		"",
	)

	// Live LLM Activity box when running
	if t.Band == view.Running {
		liveTitle := p.T("overview.live_activity", "Live LLM Activity")

		out = append(out,
			"  "+Paint(Live).Bold(true).Render(m.runGlyph(working(t))+liveTitle),
		)
		if t.CurrentAction != "" {
			actionLabel := p.T("overview.live_action", "action")
			out = append(out, fmt.Sprintf("    %-14s %s", Paint(Dim).Render(actionLabel), Paint(Accent).Render(t.CurrentAction)))
		}

		if t.CurrentThought != "" {
			thoughtLabel := p.T("overview.live_thought", "thinking")
			out = append(out, fmt.Sprintf("    %-14s %s", Paint(Dim).Render(thoughtLabel), Paint(Dim).Render(t.CurrentThought)))
		}

		if t.ToolCallCount > 0 {
			toolsLabel := p.T("overview.live_tools_count", "tool calls")
			out = append(out, fmt.Sprintf("    %-14s %s", Paint(Dim).Render(toolsLabel), Paint(Accent).Render(strconv.Itoa(t.ToolCallCount))))
		}

		if t.CurrentAction == "" && t.CurrentThought == "" {
			statusLabel := p.T("overview.state", "state")
			out = append(out,
				fmt.Sprintf("    %-14s %s", Paint(Dim).Render(statusLabel), Paint(Live).Render(m.runGlyph(working(t))+p.T("overview.running_model", "running model..."))),
				"    "+Paint(Dim).Render(p.T("overview.stream_hint", "press [6] for live timeline or [8] for raw stream output")),
			)
		}

		out = append(out, "")
	}

	// 3. Execution Summary & Outcomes
	out = append(out, "  "+Paint(Accent).Bold(true).Render(p.T("overview.execution_summary", "Execution Summary")))

	var finishedPhases []view.Entry

	for _, e := range m.entries {
		if e.Phase != "" && (e.What() == view.EntryFinished || e.What() == view.EntryFailed || e.What() == view.EntryCancelled) {
			finishedPhases = append(finishedPhases, e)
		}
	}

	if len(finishedPhases) == 0 {
		switch t.Band {
		case view.ToDo:
			out = append(out, "    "+Paint(Dim).Render(p.T("overview.not_started", "task has not been started yet (press [n] to start)")))
		case view.Running:
			out = append(out, "    "+Paint(Live).Render(m.runGlyph(working(t))+p.T("overview.in_flight", "execution in flight...")))
		default:
			out = append(out, "    "+Paint(Dim).Render(p.T("overview.no_phases", "no phase outputs recorded")))
		}
	} else {
		for _, ph := range finishedPhases {
			icon := Paint(OK).Render("✓")

			switch ph.What() {
			case view.EntryFailed:
				icon = Paint(Bad).Render("✗")
			case view.EntryCancelled:
				icon = Paint(Warn).Render("⏹")
			}

			phLine := fmt.Sprintf("    %s %s", icon, Paint(Accent).Render(ph.Phase))
			if ph.Cost > 0 {
				phLine += " " + Paint(Dim).Render(fmt.Sprintf("($%.4f)", ph.Cost))
			}

			out = append(out, phLine)

			// Clean excerpt of text output
			if ph.Text != "" {
				textLines := strings.Split(strings.TrimSpace(ph.Text), "\n")
				if !m.expandedDetail {
					for _, tl := range textLines {
						tl = strings.TrimSpace(tl)
						if tl != "" {
							out = append(out, "      "+Paint(Dim).Render("• ")+tl)
							break
						}
					}
				} else {
					for _, tl := range textLines {
						tl = strings.TrimSpace(tl)
						if tl == "" {
							continue
						}

						wrapped := splitIntoLines(tl, max(20, m.frame.Body.W-10))
						for _, wl := range wrapped {
							out = append(out, "      "+Paint(Dim).Render("• ")+wl)
						}
					}
				}
			}
		}
	}

	out = append(out, "")

	// 4. Code Changes & Impact
	diffSum := parseDiffSummary(m.diff)

	out = append(out, "  "+Paint(Accent).Bold(true).Render(p.T("overview.code_impact", "Code Impact & Changes")))
	if len(diffSum.files) > 0 || diffSum.added > 0 || diffSum.deleted > 0 {
		out = append(out, fmt.Sprintf("    %s %s %s",
			Paint(OK).Render(fmt.Sprintf("+%d", diffSum.added)),
			Paint(Bad).Render(fmt.Sprintf("-%d", diffSum.deleted)),
			Paint(Dim).Render(p.T("overview.lines_in_files", "lines across {count} file(s)", about("count", strconv.Itoa(len(diffSum.files)))))),
		)
		for i, f := range diffSum.files {
			if i >= 4 {
				rest := len(diffSum.files) - 4
				out = append(out, "      "+Paint(Dim).Render(p.P("overview.more_files", rest, "… and {n} more file", "… and {n} more files")))

				break
			}

			out = append(out, "      "+Paint(Dim).Render("• ")+Paint(Accent).Render(f))
		}
	} else {
		out = append(out, "    "+Paint(Dim).Render(p.T("overview.no_diff", "no working tree modifications recorded")))
	}

	out = append(out, "")

	// 5. Quick Actions for validation and delivery in an aligned 4x2 grid
	colWidth := 24
	row1 := "    " +
		formatActionCell("p", p.T("overview.action_pr", "create PR"), colWidth) +
		formatActionCell("u", p.T("overview.action_update_pr", "update PR"), colWidth) +
		formatActionCell("M", p.T("overview.action_merge_pr", "merge PR"), colWidth) +
		formatActionCell("X", p.T("overview.action_close_pr", "close PR"), colWidth)

	row2 := "    " +
		formatActionCell("c", p.T("overview.action_checks", "fix checks"), colWidth) +
		formatActionCell("T", p.T("overview.action_tests", "more tests"), colWidth) +
		formatActionCell("a", p.T("overview.action_feedback", "feedback"), colWidth) +
		formatActionCell("0", p.T("overview.action_diff", "diff"), colWidth)

	out = append(out,
		"  "+Paint(Accent).Bold(true).Render(p.T("overview.quick_actions", "Quick Actions to Deliver & Validate")),
		row1,
		row2,
		"",
	)

	return out
}

func formatActionCell(key, label string, width int) string {
	btn := Paint(Live).Render("[" + key + "]")
	txt := Paint(Dim).Render(" " + label)
	cell := btn + txt
	pad := max(0, width-lipgloss.Width(cell))

	return cell + strings.Repeat(" ", pad)
}
