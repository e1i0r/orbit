package ui

import (
	"strconv"
	"strings"
)

// emptyRows is the body with nothing in it, and it says which kind of
// nothing.
//
// There are three, and they want three different next moves: clear the
// filter, add a repository, write a task. The program this replaces printed
// one word for all three, and the reader's next move after reading it was to
// go looking for a task they were certain they had written.
func (m Model) emptyRows(h, w int) []string {
	p := m.opts.Words

	var lines []string

	switch typed := strings.TrimSpace(m.filter); {
	case typed != "":
		lines = []string{
			p.T("empty.filter", "Nothing matches {filter}.", about("filter", typed)), "",
			p.T("empty.clear_filter", "Press esc to clear the filter and see everything again."),
		}
	case m.board.Repos == 0:
		lines = []string{
			p.T("empty.no_repos", "No repositories under {root} yet.", about("root", m.opts.Root)), "",
			p.T("empty.add_repo", "Clone one into {root}, or start orbit in a folder that already has one.",
				about("root", m.opts.Root)),
		}
	default:
		lines = []string{
			p.T("empty.needs_you", "Nothing needs you."), "",
			p.P("empty.no_tasks", m.board.Repos,
				"{n} repository, and no tasks written against it yet.",
				"{n} repositories, and no tasks written against any of them yet.",
				about("n", strconv.Itoa(m.board.Repos))),
			p.T("empty.write_one", "Write one with `orbit new <id>`, then press n to start it."),
		}
	}

	out := []string{""}

	for i, line := range lines {
		if line == "" {
			out = append(out, "")
			continue
		}

		role := Dim
		if i == 0 {
			role = Accent
		}

		out = append(out, fit("  "+Paint(role).Render(line), w))
	}

	return fill(out, h)
}

// refusal is what a terminal narrower than the minimum gets instead of a
// crooked table: one sentence, both numbers, and the rest of the rows blank.
func (m Model) refusal() string {
	p := m.opts.Words
	w := max(m.width, 1)
	out := []string{
		fit(Paint(Warn).Render(p.T("narrow.refused", "orbit needs {need} columns.",
			about("need", strconv.Itoa(m.narrow.Need)))), w),
		fit(Paint(Dim).Render(p.T("narrow.got", "this one has {got}.",
			about("got", strconv.Itoa(m.narrow.Got)))), w),
	}

	return strings.Join(fill(out, max(m.height, 1)), "\n")
}

// page is how many rows of the list the body can show at one time.
//
// It is one less than the region's height whenever there is more list below
// the fold, and the row it gives up is spent saying so. A list that simply
// stops at the bottom of the screen is the specific bug the plan calls out:
// there is nothing on screen to distinguish "these are all of them" from
// "there are nine more".
func page(h, rows, offset int) int {
	if h <= 0 {
		return 0
	}

	if offset+h >= rows {
		return h
	}

	return h - 1
}

// bodyRow draws one line of the list, whichever of the three kinds it is.
func (m Model) bodyRow(r row, i, w int) string {
	switch {
	case r.blank:
		return ""
	case r.head:
		return m.headRow(r, i == m.cursor, w)
	}

	return m.drawRow(r, w, i == m.cursor)
}
