package ui

import (
	"fmt"
	"strconv"
	"strings"

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
		if strings.HasPrefix(l, "+++ b/") {
			f := strings.TrimPrefix(l, "+++ b/")
			if f != "" && f != "/dev/null" && !seenFiles[f] {
				seenFiles[f] = true
				sum.files = append(sum.files, f)
			}
		} else if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			sum.added++
		} else if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			sum.deleted++
		}
	}
	return sum
}

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

	// Header: Task ID, status badge and title
	statusBadge := Paint(role).Bold(true).Render("[" + word + "]")
	out = append(out,
		"",
		"  "+Paint(Accent).Bold(true).Render(t.ID)+"  "+statusBadge,
	)
	if t.Title != "" {
		out = append(out, "  "+Paint(Accent).Render(t.Title))
	}
	out = append(out, "")

	// 1. Attention/Gate banner (if waiting on user or failed)
	if role == Warn || role == Bad {
		bannerText := p.T("overview.waiting_box", "ACCIONES REQUERIDAS / ESPERANDO AL OPERADOR")
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
	if t.Engine != "" || t.Model != "" {
		engStr := t.Engine
		if t.Model != "" {
			engStr += " (" + t.Model + ")"
		}
		out = append(out, fmt.Sprintf("    %-14s %s", Paint(Dim).Render(p.T("overview.engine", "engine")), engStr))
	}
	out = append(out,
		fmt.Sprintf("    %-14s %s", Paint(Dim).Render(p.T("overview.repo", "repository")), t.Repo+" ("+t.RepoPath+")"),
		fmt.Sprintf("    %-14s %s", Paint(Dim).Render(p.T("overview.flow", "flow")), t.Flow),
		"",
	)

	// 3. Execution Summary & Outcomes
	out = append(out, "  "+Paint(Accent).Bold(true).Render(p.T("overview.execution_summary", "Execution Summary")))
	var finishedPhases []view.Entry
	for _, e := range m.entries {
		if e.Phase != "" && (e.What() == view.EntryFinished || e.What() == view.EntryFailed || e.What() == view.EntryCancelled) {
			finishedPhases = append(finishedPhases, e)
		}
	}

	if len(finishedPhases) == 0 {
		if t.Band == view.ToDo {
			out = append(out, "    "+Paint(Dim).Render(p.T("overview.not_started", "task has not been started yet (press [n] to start)")))
		} else if t.Band == view.Running {
			out = append(out, "    "+Paint(Live).Render("⚡ "+p.T("overview.in_flight", "execution in flight...")))
		} else {
			out = append(out, "    "+Paint(Dim).Render(p.T("overview.no_phases", "no phase outputs recorded")))
		}
	} else {
		for _, ph := range finishedPhases {
			icon := Paint(OK).Render("✓")
			if ph.What() == view.EntryFailed {
				icon = Paint(Bad).Render("✗")
			} else if ph.What() == view.EntryCancelled {
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
				count := 0
				for _, tl := range textLines {
					tl = strings.TrimSpace(tl)
					if tl == "" {
						continue
					}
					out = append(out, "      "+Paint(Dim).Render("• ")+tl)
					count++
					if count >= 3 {
						break
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
				out = append(out, "      "+Paint(Dim).Render(fmt.Sprintf("… and %d more files", len(diffSum.files)-4)))
				break
			}
			out = append(out, "      "+Paint(Dim).Render("• ")+Paint(Accent).Render(f))
		}
	} else {
		out = append(out, "    "+Paint(Dim).Render(p.T("overview.no_diff", "no working tree modifications recorded")))
	}
	out = append(out, "")

	// 5. Quick Actions for validation
	out = append(out,
		"  "+Paint(Accent).Bold(true).Render(p.T("overview.quick_actions", "Quick Actions to Validate")),
		"    "+Paint(Live).Render("[0]")+Paint(Dim).Render(" "+p.T("overview.action_diff", "diff changes"))+"    "+
			Paint(Live).Render("[7]")+Paint(Dim).Render(" "+p.T("overview.action_report", "full report"))+"    "+
			Paint(Live).Render("[6]")+Paint(Dim).Render(" "+p.T("overview.action_timeline", "timeline"))+"    "+
			Paint(Live).Render("[d]")+Paint(Dim).Render(" "+p.T("overview.action_read", "mark read")),
		"",
	)

	return out
}
