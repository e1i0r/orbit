package settings

// View is the screen drawn: the table, the pills of every dial, and the line
// at the bottom saying how to leave.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// View is the whole screen, already cut to the room it was given.
func (s State) View(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	p := e.Words
	head := []string{
		"",
		"  " + theme.Paint(theme.Accent).Bold(true).Render(p.T("settings.title", "Settings")),
		"  " + theme.Paint(theme.Dim).Render(p.T("settings.subtitle", "changes take effect immediately")),
		"",
	}

	var body []string

	rows := s.Rows(e)
	for i, r := range rows {
		if headed(rows, i) {
			body = append(body, cells.Fit("  "+s.heading(rows, r.Group), w))
		}

		if s.folded[r.Group] {
			continue
		}

		lines := s.row(r, s.head == "" && i == s.sel, w)
		if r.Group != "" {
			lines = railed(lines, lastOf(rows, i), w)
		}

		body = append(body, lines...)
	}

	foot := []string{cells.Fit("  "+theme.Paint(theme.Dim).Render(s.waysOut(e)), w)}

	return cells.Fill(framed(head, body, foot, h, s.off), h)
}

// heading is a group's name, with the sign that says whether it is open
// and, folded, how many settings it hides. The cursor standing on it is
// the heading drawn in the colour a chosen row's name is.
func (s State) heading(rows []Row, group string) string {
	role := theme.Accent
	if s.head == group {
		role = theme.Live
	}

	text := "▾ " + group
	if s.folded[group] {
		text = fmt.Sprintf("▸ %s (%d)", group, counted(rows, group))
	}

	return theme.Paint(role).Bold(true).Render(text)
}

// railed is a row of a group with the line down the side of it, drawn in
// the margin under the heading's sign. The cursor's pointer stands on the
// line, and the blank under a group's last row has none: that is where the
// group ends.
func railed(lines []string, last bool, w int) []string {
	rail := theme.Paint(theme.Dim).Render("│")
	out := make([]string, len(lines))

	for i, l := range lines {
		switch {
		case i == 0 && strings.HasPrefix(l, "    "):
			out[i] = cells.Fit("  "+rail+l[3:], w)
		case i == 0:
			out[i] = l
		case i == len(lines)-1 && last:
			out[i] = ""
		case l == "":
			out[i] = "  " + rail
		default:
			out[i] = cells.Fit("  "+rail+l[3:], w)
		}
	}

	return out
}

// row is one setting drawn: its name, its pills, and the sentence under it.
func (s State) row(r Row, chosen bool, w int) []string {
	mark, keyRole, aboutRole := "    ", theme.Accent, theme.Dim
	if chosen {
		mark = "  " + theme.Paint(theme.Live).Bold(true).Render("▸ ")
		keyRole, aboutRole = theme.Live, theme.Accent
	}

	var pills []string

	switch {
	case chosen && s.editing:
		pills = append(pills, theme.Paint(theme.Accent).Render(s.typed)+theme.Paint(theme.Sel).Render(" "))
	case len(r.Options) == 0:
		// A row with nothing to offer shows what it holds. There is no
		// list of chat ids or of dollar figures worth putting under a
		// cursor, and a row that drew neither pills nor a value was a name
		// with an empty space after it — which reads as a setting that is
		// broken rather than one that is typed into.
		pills = append(pills, theme.Paint(theme.Sel).Bold(true).Render(" "+written(r.Val)+" "))
	default:
		for i, opt := range r.Options {
			if opt == r.Val {
				pills = append(pills, theme.Paint(theme.Sel).Bold(true).Render(" ● "+r.Label(i)+" "))
				continue
			}

			pills = append(pills, theme.Paint(theme.Dim).Render(" "+r.Label(i)+" "))
		}
	}

	name := theme.Paint(keyRole).Bold(true).Render(cells.PadRight(r.Key, nameWidth))
	head := mark + name + "  " + strings.Join(pills, " ")
	about := "      " + theme.Paint(aboutRole).Render(r.About)

	return []string{cells.Fit(head, w), cells.Fit(about, w), ""}
}

// nameWidth is the column the settings' names are padded to: the longest of
// them, so that every dial on the screen starts at the same cell. A name
// longer than the column does not truncate, it pushes — and one row's pills
// out of line with the rest is the whole table looking crooked.
const nameWidth = 16

// PillsAt is the column a row's options start at: the pointer's margin, the
// name column, and the two spaces after it.
//
// A door because the window measures a click against it. It was a 20 written
// out in the hit-tester while the drawing added its own three numbers up, and
// the two agreed only for as long as nobody touched either — the name column
// widened by two for budget-workspace, and every click on a pill would have
// landed two cells to the left of the one it was on.
const PillsAt = 4 + nameWidth + 2

// written is a value as a row without a dial shows it. A setting that holds
// nothing says so in a word: an empty space beside a name is a row a reader
// cannot tell from a broken one.
func written(val string) string {
	if val == "" {
		return "—"
	}

	return val
}

// waysOut is the line at the bottom, which says different things while a row
// is being typed into.
func (s State) waysOut(e Env) string {
	p, k := e.Words, e.Keys
	if s.editing {
		return p.T("settings.ways_out_edit", "{open} save · {back} cancel",
			words.Arg{Name: "open", Value: k.Open.Help().Key},
			words.Arg{Name: "back", Value: k.Back.Help().Key})
	}

	if s.head != "" {
		return p.T("settings.ways_out_fold", "{open} open or close · {up_down} move · {back} back",
			words.Arg{Name: "open", Value: k.Open.Help().Key},
			words.Arg{Name: "up_down", Value: k.Up.Help().Key + k.Down.Help().Key},
			words.Arg{Name: "back", Value: k.Back.Help().Key})
	}

	return p.T("settings.ways_out", "{open} edit · {up_down} move · {back} back",
		words.Arg{Name: "open", Value: k.Open.Help().Key},
		words.Arg{Name: "up_down", Value: k.Up.Help().Key + k.Down.Help().Key},
		words.Arg{Name: "back", Value: k.Back.Help().Key})
}
