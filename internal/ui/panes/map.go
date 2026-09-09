package panes

// The repository, and where this task went in it.
//
// The browser draws this as a honeycomb, and a terminal should not try.
// A hexagon a reader can read needs about nine columns by five rows, so at
// the hundred columns this window is checked at, a level would fit eight
// cells and none of them would have room for its name — a panel that says
// "int…" and has to be hovered is worse than the list it replaced.
//
// What travels is not the shape, it is the answer: *where did this task go*.
// A terminal says that best as the tree itself, lit where the work happened
// and weighted by how much, which is the same two facts the comb draws with
// colour and heat. Both read verb.Grow, so they cannot disagree about what
// is in a repository.
//
// Only the branches the task touched are drawn. A repository has hundreds of
// directories and this pane has forty rows; a map of everything is a map
// nobody scrolls to the end of, and the question here is what changed.

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/verb"
)

// bar is how many cells the weight is drawn across.
const bar = 10

// Map is the tree of the checkout, marked where this task changed it.
func Map(e Env) []string {
	p, m := e.Words, e.Shape

	switch {
	case m.Asking && !m.Read:
		return []string{e.Spinner + theme.Paint(theme.Live).Render(
			p.T("map.reading", "reading the repository…"))}
	case !m.Read:
		return []string{theme.Paint(theme.Dim).Render(p.T("map.not_yet", "nothing read yet"))}
	case m.Missing:
		// Said as a state and not as a failure. A checkout is taken away by
		// hand and cleaned up after a task ends, and both are ordinary; the
		// task's own record is still there to read.
		return []string{theme.Paint(theme.Dim).Render(p.T("map.no_checkout",
			"this task has no checkout any more, so there is nothing to map"))}
	case m.Failed != "":
		return []string{theme.Paint(theme.Bad).Render(p.T("map.failed",
			"the repository could not be read: {err}", about("err", m.Failed)))}
	case m.Tree.Changed == 0:
		return []string{theme.Paint(theme.Dim).Render(p.T("map.no_changes",
			"this task has changed nothing in this checkout yet"))}
	}

	head := p.T("map.head", "{files} files · {lines} lines, in {name}",
		about("files", strconv.Itoa(m.Tree.Changed)),
		about("lines", strconv.Itoa(m.Tree.Lines)),
		about("name", e.Task.Repo))

	rows := []string{theme.Paint(theme.Dim).Render(head), ""}

	return append(rows, branch(m.Tree, m.Tree.Lines, "")...)
}

// branch is one cell's touched children, and theirs.
//
// The stem is carried down as a string rather than rebuilt from a depth,
// because what a row needs is not how deep it is but which of its ancestors
// were the last of their level — that is the difference between a line that
// continues past a row and a space.
func branch(at verb.Cell, most int, stem string) []string {
	var (
		lit  []verb.Cell
		rows []string
	)

	for _, c := range at.Cells {
		if c.Changed > 0 {
			lit = append(lit, c)
		}
	}

	for i, c := range lit {
		last := i == len(lit)-1

		elbow, under := "├─ ", "│  "
		if last {
			elbow, under = "└─ ", "   "
		}

		rows = append(rows, row(c, most, stem+elbow))
		rows = append(rows, branch(c, most, stem+under)...)
	}

	return rows
}

// row is one cell: where it sits, what it is called, how heavy it is, and
// what the task did to it.
func row(c verb.Cell, most int, stem string) string {
	leaf := len(c.Cells) == 0

	name := c.Name
	if leaf {
		name = theme.Text(theme.Primary).Render(name)
	} else {
		name = theme.Paint(theme.Accent).Render(name)
	}

	// The name column is padded by cell width and not by byte length: a
	// directory called ñandú is five cells wide and seven bytes long, and
	// padding by bytes puts its weight bar two columns left of everybody
	// else's. lipgloss.Width is what the rest of this package measures with.
	line := theme.Paint(theme.Dim).Render(stem) + name
	if pad := 34 - lipgloss.Width(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}

	said := strconv.Itoa(c.Lines)
	if !leaf {
		said = strconv.Itoa(c.Changed) + " · " + said
	}

	return line + " " + weight(c.Lines, most) + " " +
		theme.Paint(theme.Dim).Render(said)
}

// weight is how much of the change is under this cell, as a bar.
//
// Against the busiest cell of the whole tree rather than against its own
// level, so the bars mean one thing everywhere in the drawing: a row half
// full is half of the task's work, wherever it appears.
func weight(lines, most int) string {
	if most <= 0 {
		return strings.Repeat("░", bar)
	}

	full := lines * bar / most
	if full == 0 && lines > 0 {
		full = 1
	}

	return theme.Paint(theme.Accent).Render(strings.Repeat("█", full)) +
		theme.Paint(theme.Dim).Render(strings.Repeat("░", bar-full))
}
