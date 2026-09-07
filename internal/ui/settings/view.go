package settings

// View is the screen drawn: the table, the pills of every dial, and the line
// at the bottom saying how to leave.

import (
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
	out := []string{
		"",
		"  " + theme.Paint(theme.Accent).Bold(true).Render(p.T("settings.title", "Settings")),
		"  " + theme.Paint(theme.Dim).Render(p.T("settings.subtitle", "changes take effect immediately")),
		"",
	}

	for i, r := range s.Rows(e) {
		out = append(out, s.row(r, i == s.sel, w)...)
	}

	return cells.Fill(append(out, cells.Fit("  "+theme.Paint(theme.Dim).Render(s.waysOut(e)), w)), h)
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
	default:
		for i, opt := range r.Options {
			if opt == r.Val {
				pills = append(pills, theme.Paint(theme.Sel).Bold(true).Render(" ● "+r.Label(i)+" "))
				continue
			}

			pills = append(pills, theme.Paint(theme.Dim).Render(" "+r.Label(i)+" "))
		}
	}

	name := theme.Paint(keyRole).Bold(true).Render(cells.PadRight(r.Key, 14))
	head := mark + name + "  " + strings.Join(pills, " ")
	about := "      " + theme.Paint(aboutRole).Render(r.About)

	return []string{cells.Fit(head, w), cells.Fit(about, w), ""}
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

	return p.T("settings.ways_out", "{open} edit · {up_down} move · {back} back",
		words.Arg{Name: "open", Value: k.Open.Help().Key},
		words.Arg{Name: "up_down", Value: k.Up.Help().Key + k.Down.Help().Key},
		words.Arg{Name: "back", Value: k.Back.Help().Key})
}
