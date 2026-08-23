package ui

// The window's furniture: the rule, the header line and the key bar. None of
// these is a list of tasks, and every one of them is a pure function of the
// width it is given and the model it is called on — nothing here asks the
// terminal anything. The fourth region, the activity band, is in band.go.

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/board"
)

const (
	// pipOn and pipOff are the standing switch, as one cell. They are
	// glyphs rather than the words on and off because the word beside them
	// already says which switch it is, and a filled circle is read without
	// being read.
	pipOn  = "●"
	pipOff = "○"

	// dot joins the pieces of a sentence the window assembles out of facts
	// — an id, a phase, a model. A comma would imply somebody wrote the
	// sentence.
	dot = " · "

	// headerGap is the space between two of the header's right-hand fields
	// and hintGap the space between two hints in the key bar. Four and two,
	// for the reason layout's gap is two: one space reads as a wide space
	// inside a field rather than as a boundary between two.
	headerGap = "    "
	hintGap   = "  "

	// The rule's two greys. This is the only pair in the window and the
	// only place the terminal's background colour changes anything: the
	// rule is furniture, so it has to sit below the text on both, and one
	// grey cannot do that.
	ruleOnLight = "250"
	ruleOnDark  = "238"
)

// rule is the horizontal line between two regions.
//
// It is where m.dark is spent. The value arrives as a tea.BackgroundColorMsg
// answering the tea.RequestBackgroundColor that Init sends, which is the only
// way this program ever learns it — lipgloss.HasDarkBackground and
// compat.AdaptiveColor both read the terminal from wherever they are called,
// and inside a render that is a blocking read in the middle of a frame.
func (m Model) rule(w int) string {
	if w <= 0 {
		return ""
	}
	shade := lipgloss.LightDark(m.dark)(lipgloss.Color(ruleOnLight), lipgloss.Color(ruleOnDark))
	return lipgloss.NewStyle().Foreground(shade).Render(strings.Repeat("─", w))
}

// headerLine is the program, the folder it was opened on, and the three
// standing facts.
//
// Two things are given up as the terminal narrows, and the order they go in
// is the whole decision. The root path goes first, from the front, because a
// reader who opened the folder knows which folder it is and the tail of a
// path says more than its head. Then the right-hand fields go from the
// right, which is why headerFields puts them in the order it does: a
// repository count is a number a reader can get elsewhere, the pip is a
// setting they just changed themselves, and the unread pair is the brake
// that stops tasks from starting. Losing the brake to fit either of the
// other two would be losing the one field on this line that changes what
// happens next.
func (m Model) headerLine(w int) string {
	fields := m.headerFields()
	for {
		right := strings.Join(fields, headerGap)
		if line, ok := m.headerLeft(w-lipgloss.Width(right), right != ""); ok {
			gap := w - lipgloss.Width(line) - lipgloss.Width(right)
			return line + strings.Repeat(" ", gap) + right
		}
		if len(fields) == 0 {
			return fit(m.name(), w)
		}
		fields = fields[:len(fields)-1]
	}
}

// headerLeft is the program and the folder, shortened from the front of the
// path until both fit in the cells the fields left over, or refused when
// even the program's own name and a bare root will not.
//
// The path is cut at the front and marked with an ellipsis, which is the
// opposite of what fit does everywhere else in this file and is right here
// for one reason: the last two segments of a path identify it and the first
// two rarely do.
func (m Model) headerLeft(w int, spaced bool) (string, bool) {
	if spaced {
		w--
	}
	name := m.name()
	for root := m.opts.Root; ; {
		line := name
		if root != "" {
			line = name + "  " + Paint(Dim).Render(root)
		}
		if lipgloss.Width(line) <= w {
			return line, true
		}
		if root == "" {
			return "", false
		}
		root = shorten(root)
	}
}

// name is the program's own name, which is never given up: a window that
// cannot say what it is, is worse than a window showing less.
func (m Model) name() string { return " " + Paint(Accent).Render("orbit") }

// shorten drops one leading path segment and marks the cut, and returns ""
// once there is nothing left worth showing.
func shorten(root string) string {
	trimmed := strings.TrimPrefix(root, "…/")
	if i := strings.IndexByte(trimmed, '/'); i >= 0 && trimmed[i+1:] != "" {
		return "…/" + trimmed[i+1:]
	}
	return ""
}

// headerFields are the standing facts, in the order they are given up
// backwards: unread is never dropped while any field is shown, the pip goes
// before it, and the repository count goes first.
func (m Model) headerFields() []string {
	p := m.opts.Words
	unread, limit := board.Unread(m.board), m.unreadCap()

	brake := Dim
	if limit > 0 && unread >= limit {
		brake = Warn
	}
	fields := []string{Paint(brake).Render(p.T("header.unread", "unread {n}/{cap}",
		about("n", strconv.Itoa(unread)), about("cap", strconv.Itoa(limit))))}

	pip, role := pipOff, Dim
	if m.autopilotOn() {
		pip, role = pipOn, Live
	}
	fields = append(fields, Paint(Dim).Render(p.T("header.autopilot", "autopilot"))+" "+Paint(role).Render(pip))
	return append(fields, Paint(Dim).Render(p.P("header.repos", m.board.Repos, "{n} repo", "{n} repos")))
}

// barLine is what can be pressed right now.
//
// It drops whole hints from the right rather than truncating them, because
// half a hint is a key a reader has to guess the rest of. Help and quit are
// never dropped: they are how a reader who is lost gets out, and a bar that
// drops them is a bar that fails exactly when it is needed.
func (m Model) barLine(w int) string {
	tail := Paint(Dim).Render("[" + m.keys.Help.Help().Key + "] [" + m.keys.Quit.Help().Key + "]")
	hints := m.hints()
	for {
		line := " " + strings.Join(append(hints, tail), hintGap)
		if lipgloss.Width(line) <= w || len(hints) == 0 {
			return fit(line, w)
		}
		hints = hints[:len(hints)-1]
	}
}

// hints are the bar's entries, in the order they are given up backwards.
//
// Everything about the task under the cursor comes from Affordances, so a
// key the bar offers is a key that will not be refused when it is pressed.
// The bar shows what can be done; the menu, one level down, shows what
// cannot and why.
func (m Model) hints() []string {
	var out []string
	r, ok := m.selected()
	if ok {
		out = append(out, hint("↑↓", m.opts.Words.T("key.move", "move")), hintFor(m.keys.Open))
	}
	out = append(out, hintFor(m.keys.New))
	if ok && !r.head {
		for _, a := range m.keys.Affordances(r.task, m.conditions()) {
			if a.OK && a.Key.Help().Key != m.keys.Open.Help().Key {
				out = append(out, hintFor(a.Key))
			}
		}
	}
	return append(out, hintFor(m.keys.Filter))
}

// hintFor is one binding as the bar prints it, and hint is the same for the
// arrow pair, which is two keys with one meaning and so has no binding of
// its own.
func hintFor(b key.Binding) string { return hint(b.Help().Key, b.Help().Desc) }

func hint(glyph, desc string) string {
	return Paint(Accent).Render("["+glyph+"]") + " " + Paint(Dim).Render(desc)
}
