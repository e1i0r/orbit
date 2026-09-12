package ui

// Where the window and the command palette meet.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/palette"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// paletteEnv is the world the palette was written against. The commands are
// translated here: the screen is handed sentences, not the printer and a
// closure to call it with.
func (m Model) paletteEnv() palette.Env {
	p := m.opts.Words

	cmds := make([]palette.Command, 0, len(m.opts.Commands))

	for _, c := range m.opts.Commands {
		row := palette.Command{
			Name:       c.Name,
			Args:       c.Args,
			Refused:    c.Refused,
			NeedsArgs:  c.NeedsArgs,
			AboutATask: c.AboutATask,
		}

		for _, kid := range c.Children {
			about := ""
			if kid.About != nil {
				about = kid.About(p)
			}

			row.Children = append(row.Children, palette.Child{Name: kid.Name, About: about})
		}

		if c.About != nil {
			row.About = c.About(p)
		}

		if c.Because != nil {
			row.Because = c.Because(p)
		}

		cmds = append(cmds, row)
	}

	return palette.Env{Words: p, Keys: m.keys, Frame: m.frame, Commands: cmds}
}

// tookPalette does what the line asked the window for.
func (m Model) tookPalette(next palette.State, out palette.Out) (tea.Model, tea.Cmd) {
	m.palette = next

	if out.Said != "" {
		m = m.say(out.Said)
	}

	if out.Run == "" {
		return m, nil
	}

	return m.launchNamed(out.Run, palette.Args(out.Line))
}

// launchNamed runs the command of that name, through the same port the
// keyboard and the pointer both reach. A refusal comes back verbatim and is
// said in the band, because the window is a keyboard in front of the
// commands and not a second copy of their rules.
func (m Model) launchNamed(name string, args []string) (tea.Model, tea.Cmd) {
	for _, c := range m.opts.Commands {
		if c.Name == name {
			return m.launch(c, args)
		}
	}

	return m, nil
}

// openPalette brings the line up empty.
func (m Model) openPalette() Model {
	m.palette = palette.Open()

	return m
}

// openPaletteWith brings it up with something already on the line, for the
// one caller that knows what the reader is about to type: the menu, whose
// task commands need an id it has no way of supplying.
func (m Model) openPaletteWith(typed string) Model {
	m.palette = palette.OpenWith(typed)

	return m
}

// paletteKey hands one keystroke to the line.
func (m Model) paletteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, out := m.palette.Key(msg, m.paletteEnv())

	return m.tookPalette(next, out)
}

// chooseCommand runs the command a click landed on.
func (m Model) chooseCommand(name string) (tea.Model, tea.Cmd) {
	next, out := m.palette.Choose(name, m.paletteEnv())

	return m.tookPalette(next, out)
}

// pick moves the palette's selection, for the wheel.
func (m Model) pick(d int) Model {
	m.palette = m.palette.Wheel(d, m.paletteEnv())

	return m
}

// paletteRows is the list drawn.
func (m Model) paletteRows(h, w int) []string {
	return m.palette.View(h, w, m.paletteEnv())
}

// paletteInputLine is the line itself, where the key bar sits while the
// palette owns the keyboard.
func (m Model) paletteInputLine(w int) string {
	return m.palette.Line(w, m.paletteEnv())
}

// hitPalette is what the list has at that cell.
func (m Model) hitPalette(x, y int) point.Target {
	return m.palette.Hit(x, y, m.paletteEnv())
}
