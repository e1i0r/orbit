package ui

import (
	"fmt"
	"strings"
)

type artifactInfo struct {
	path      string
	additions int
	deletions int
	isNew     bool
	isDeleted bool
}

// artifactsLines renders Pane 8: Files and artifacts changed in this task.
func (m Model) artifactsLines() []string {
	p := m.opts.Words
	if m.diffErr != nil {
		return []string{"  " + Paint(Bad).Render(m.diffErr.Error())}
	}
	if !m.diffKnown || strings.TrimSpace(m.diff) == "" {
		return []string{"", "  " + Paint(Dim).Render(p.T("artifacts.empty", "no files or artifacts modified yet"))}
	}

	files := parseArtifacts(m.diff)
	if len(files) == 0 {
		return []string{"", "  " + Paint(Dim).Render(p.T("artifacts.empty", "no files or artifacts modified yet"))}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("artifacts.title", "Modified Artifacts & Files ({n})",
			about("n", fmt.Sprintf("%d", len(files))))),
		"",
	}

	for _, f := range files {
		tag := Paint(Dim).Render("M")
		if f.isNew {
			tag = Paint(OK).Render("A")
		} else if f.isDeleted {
			tag = Paint(Bad).Render("D")
		}
		stats := fmt.Sprintf("+%d -%d", f.additions, f.deletions)
		line := fmt.Sprintf("    [%s] %-40s  %s", tag, Paint(Accent).Render(f.path), Paint(Dim).Render(stats))
		out = append(out, line)
	}

	return out
}

func parseArtifacts(diffText string) []artifactInfo {
	var list []artifactInfo
	var current *artifactInfo

	for _, line := range strings.Split(diffText, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			if current != nil {
				list = append(list, *current)
			}
			parts := strings.Fields(line)
			path := ""
			if len(parts) >= 4 {
				path = strings.TrimPrefix(parts[3], "b/")
			}
			current = &artifactInfo{path: path}
			continue
		}
		if current == nil {
			continue
		}
		if strings.HasPrefix(line, "new file mode") {
			current.isNew = true
		} else if strings.HasPrefix(line, "deleted file mode") {
			current.isDeleted = true
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			current.additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			current.deletions++
		}
	}
	if current != nil {
		list = append(list, *current)
	}
	return list
}
