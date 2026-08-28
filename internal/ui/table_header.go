package ui

import (
	"strings"
)

// tableHeader draws the column labels aligned with the layout plan.
func (m Model) tableHeader(w int) string {
	p := m.opts.Words
	fields := []struct {
		cells int
		text  string
		right bool
	}{
		{m.plan.Repo, p.T("board.col_repo", "REPO"), false},
		{m.plan.ID, p.T("board.col_id", "ID"), false},
		{m.plan.Title, p.T("board.col_title", "TASK"), false},
		{m.plan.State, p.T("board.col_state", "STATUS"), false},
		{m.plan.Model, p.T("board.col_model", "MODEL"), false},
		{m.plan.Elapsed, p.T("board.col_elapsed", "TIME"), true},
	}

	var parts []string

	for _, f := range fields {
		if f.cells <= 0 {
			continue
		}

		cell := pad(f.text, f.cells, f.right)
		parts = append(parts, Paint(Dim).Bold(true).Render(cell))
	}

	line := "  " + strings.Join(parts, strings.Repeat(" ", columnGap))

	return fit(line, w)
}
