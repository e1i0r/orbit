package ui

import (
	"strconv"
	"strings"
)

// cockpitSplitRows renders the two-column cockpit layout:
// - Left: Task list with band headings and cursor selection
// - Right: Live detail view for the selected task with 11 panes
func (m Model) cockpitSplitRows(h, w int, all []row) []string {
	listW := min(36, max(30, w*35/100))
	detailW := w - listW - 1

	// 1. Left List Rows
	leftOut := make([]string, 0, h)
	shown := page(h, len(all), m.offset)
	for i := m.offset; i < len(all) && len(leftOut) < shown; i++ {
		leftOut = append(leftOut, m.bodyRow(all[i], i, listW))
	}
	if hidden := len(all) - m.offset - len(leftOut); hidden > 0 {
		more := m.opts.Words.P("body.more", hidden, "… and {n} more", "… and {n} more",
			about("n", strconv.Itoa(hidden)))
		leftOut = append(leftOut, fit(strings.Repeat(" ", gutter)+Paint(Dim).Render(more), listW))
	}
	leftLines := fill(leftOut, h)

	// 2. Right Detail Rows
	var rightLines []string
	selRow, ok := m.selected()
	if ok && !selRow.head && !selRow.blank {
		prevDetail := m.detail
		m.detail = selRow.task.ID
		rightLines = m.detailRows(h, detailW)
		m.detail = prevDetail
	} else {
		rightLines = fill([]string{
			"",
			"  " + Paint(Dim).Render(m.opts.Words.T("detail.select", "Select a task on the left to inspect its cockpit.")),
		}, h)
	}

	// 3. Join Left + Divider + Right
	div := Paint(Dim).Render("│")
	joined := make([]string, h)
	for i := 0; i < h; i++ {
		l := ""
		if i < len(leftLines) {
			l = fit(leftLines[i], listW)
		} else {
			l = strings.Repeat(" ", listW)
		}
		r := ""
		if i < len(rightLines) {
			r = fit(rightLines[i], detailW)
		}
		joined[i] = l + div + r
	}
	return joined
}
