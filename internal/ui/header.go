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
// path says more than its head. Then the right-hand fields go from the front
// of the list, which is why headerFields puts them in the order it does: the
// upgrade notice names a command that will still be there tomorrow, a
// repository count is a number a reader can get elsewhere, the pip is a
// setting they just changed themselves, and the unread pair is the brake
// that stops tasks from starting. Losing the brake to fit any of the others
// would be losing the one field on this line that changes what happens
// next.
func (m Model) headerLine(w int) string {
	line, _ := m.headerLayout(w)
	return line
}

// headerLayout is that line, drawn, and where it put each thing a reader can
// click.
//
// The zones come out of the pass that builds the line, which is the whole
// point of returning them. Column numbers of hitHeader's own — under 10 is
// the badge, 28 to 44 is Running — are written down once against a header
// that is laid out again with every change to this line. Measured against
// what is actually drawn, the left third of Running filters by To Do and
// Needs You filters by Running; and because the badge of the queue being
// filtered on is two cells wider than the others, selecting one moves every
// band after it a second time without hitHeader hearing about it.
func (m Model) headerLayout(w int) (string, []headerZone) {
	fields := m.headerFields()
	for {
		right := strings.Join(fieldTexts(fields), headerGap)
		if line, zones, ok := m.headerLeft(w-lipgloss.Width(right), right != ""); ok {
			gap := w - lipgloss.Width(line) - lipgloss.Width(right)

			return line + strings.Repeat(" ", gap) + right,
				append(zones, placeFields(fields, lipgloss.Width(line)+gap)...)
		}

		if len(fields) == 0 {
			line := fit(m.name(), w)

			return line, []headerZone{{
				target: Target{Kind: TargetHeaderField, Field: "orbit"},
				w:      lipgloss.Width(line),
			}}
		}

		// The field given up is the first, not the last. Taking the last
		// was taking the brake: it is appended after the three chips, so a
		// header one cell too narrow dropped the field that says why
		// nothing is starting and kept an upgrade notice for another
		// forty-five.
		fields = fields[1:]
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
func (m Model) headerLeft(w int, spaced bool) (string, []headerZone, bool) {
	if spaced {
		w--
	}

	name := m.name()
	zones := []headerZone{{
		target: Target{Kind: TargetHeaderField, Field: "orbit"},
		w:      lipgloss.Width(name),
	}}

	// 1. Try full queue badges line if it fits
	if badges := m.queueBadges(); len(badges) > 0 {
		full := name + "  " + strings.Join(badgeTexts(badges), " ")
		if lipgloss.Width(full) <= w {
			return full, append(zones, placeBadges(badges, lipgloss.Width(name)+2)...), true
		}
	}

	for root := m.opts.Root; ; {
		line := name
		if root != "" {
			line = name + "  " + Chrome().Render(root)
		}

		if lipgloss.Width(line) <= w {
			return line, zones, true
		}

		if root == "" {
			return "", nil, false
		}

		root = shorten(root)
	}
}

// name is the program's own name badge: the mark from the logo, which is a
// body with rings around it, and the word.
//
// It is not "[orbit]": the brackets do the job the pill's own background
// already does — saying where the badge starts and stops — and everything
// else on this line that a reader can click is a pill without them.
//
// The glyph and the two brackets are the same three cells wide, so the badge
// is nine either way and hitHeader's columns land where they did. That is
// luck rather than design, and the next change to this string has to check
// it: the click that resets every filter is placed by a number
// written down in target.go, not by measuring what was drawn here.
func (m Model) name() string {
	const (
		fg = "#FFFFFF"
		bg = "#0F766E"
	)

	if m.showingEverything() {
		return PillSelected("◉ orbit", fg, bg)
	}

	return Pill("◉ orbit", fg, bg)
}

// showingEverything reports whether the board is holding nothing back, which
// is the state clicking the name badge puts it in: no queue chosen, no
// repository chosen, nothing typed in the search.
//
// The badge is drawn as selected exactly then. A queue badge lights up when
// its queue is the one being shown, and until now the badge that means "all
// of them" was the one arrangement of the board with nothing lit at all.
func (m Model) showingEverything() bool {
	return m.queueFilter == nil && m.repoFilter == "" && m.filter == ""
}

// shorten drops one leading path segment and marks the cut, and returns ""
// once there is nothing left worth showing.
func shorten(root string) string {
	trimmed := strings.TrimPrefix(root, "…/")
	if i := strings.IndexByte(trimmed, '/'); i >= 0 && trimmed[i+1:] != "" {
		return "…/" + trimmed[i+1:]
	}

	return ""
}

// headerFields are the standing facts with emoji chips in Monokai theme.
//
// Three of them carry a name, and those are the three a click on the header
// opens something for. The upgrade notice and the brake are read and not
// pressed: one names a command to run, the other names a limit to clear, and
// neither has a screen behind it to open.
func (m Model) headerFields() []headerField {
	p := m.opts.Words

	var fields []headerField

	// Upgrade available notice (pastel mint on deep emerald)
	if m.upgradeAvailable != "" {
		ver := "v" + strings.TrimPrefix(m.upgradeAvailable, "v")
		notice := p.T("header.upgrade_notice", "{version} available · orbit upgrade",
			about("version", ver))
		fields = append(fields, headerField{text: Pill(" ✨ "+notice+" ", inkUpgrade.fg, inkUpgrade.bg)})
	}

	// Repos chip.
	//
	// The one number here that is not a count of tasks, and the reason the
	// band counts beside it can be read as tasks at all: they count rows,
	// and a row is a task however many of these it reaches into. A chip
	// saying how many tasks there are was tried here and taken out again —
	// it cost the band counts their place at a hundred cells, to repeat a
	// number the status line already gives and the bands already add up to.
	reposText := p.P("header.repos", m.board.Repos, "{n} repo", "{n} repos")
	fields = append(fields, headerField{"repos", Chrome().Render("📦 " + reposText)})

	// Model / knob chip. With no knob set this is still the engine that
	// answers, and not the literal "claude" it used to print.
	chip, ink := m.knobChip(), Paint(Accent)
	if chip == "" {
		chip, ink = m.dialEngine(""), Chrome()
	}

	fields = append(fields, headerField{"engine", ink.Render("🧠 " + chip)})

	// Quota chip
	//
	// Named, so the pointer can reach it: what is behind the percentage is
	// several windows of several engines, none of which fits on this line,
	// and the quota screen is where all of it is written down. It sits after
	// the engine because it is that engine's number.
	if q := m.quotaChip(); q != "" {
		fields = append(fields, headerField{"quota", Chrome().Render("⏳ " + q)})
	}

	// Language chip
	lang := p.T("header.lang_badge", "EN")
	fields = append(fields, headerField{"lang", Chrome().Render("🌐 " + lang)})

	fields = append(fields, m.runningField(p)...)
	fields = append(fields, m.brakeField(p)...)

	// Unread brake warning (shown when brake is engaged)
	unread := board.Unread(m.board)
	if m.atUnreadCap(unread) {
		brakeText := p.T("header.unread_brake", "brake ({n} unread)",
			about("n", strconv.Itoa(unread)))
		fields = append(fields, headerField{text: Paint(Warn).Render("⚠️ " + brakeText)})
	}

	return fields
}

// hints are the bar's entries, in the order they are given up backwards.
//
// Everything about the task under the cursor comes from Affordances, so a
// key the bar offers is a key that will not be refused when it is pressed.
// The bar shows what can be done; the menu, one level down, shows what
// cannot and why.
func (m Model) hints() []barHint {
	switch m.screen {
	case screenDetail:
		return m.detailHints()
	case screenStart:
		return m.startHints()
	}

	var out []barHint

	r, ok := m.selected()
	if ok {
		out = append(out, hint("↑↓", m.opts.Words.T("key.move", "move")), hintFor(m.keys.Open))
	}

	out = append(out, hintFor(m.keys.Start))
	if ok && !r.head {
		for _, a := range m.keys.Affordances(r.task, m.conditions(r.task)) {
			if a.OK && a.Key.Help().Key != m.keys.Open.Help().Key {
				out = append(out, hintFor(a.Key))
			}
		}
	}

	// On the bar and not only in the help overlay: a key a reader never
	// sees is a key they never press.
	return append(out, hintFor(m.keys.Supervisor), hintFor(m.keys.Flows), hintFor(m.keys.Filter))
}

// hintFor is one binding as the bar prints it: the glyph a reader sees, the
// description beside it, and the keystroke a click on it would send.
//
// The keystroke is the binding's own first key rather than the glyph, and
// the two are not the same string — ⏎ is drawn and enter is pressed. Taking
// it from the binding is also what keeps a clicked hint and a pressed key on
// one path: both arrive at the board's map as the same keystroke, so a verb
// cannot be reachable by one and not by the other.
func hintFor(b key.Binding) barHint {
	h := hint(b.Help().Key, b.Help().Desc)
	h.key = string(firstKey(b))

	return h
}

// hint is the same for a pair of keys with one meaning — the arrows — which
// has no binding of its own and so no single keystroke to send. It is drawn
// and it is inert.
func hint(glyph, desc string) barHint {
	return barHint{text: Paint(Accent).Render("["+glyph+"]") + " " + Chrome().Render(desc)}
}

// hintKey is a hint whose glyph is the whole of the keystroke it sends.
//
// Some of what the task view answers is matched by the letter inside
// detailKey rather than by a binding in m.keys, and drawn with hint those
// were drawn as keys and clicked as nothing: [m] tab menu, [v] md / raw and
// [e] expand did what pressing them does and nothing at all from the
// pointer. A hint that names a key a reader can press is a hint they can
// click, and the click sends that key.
func hintKey(glyph, desc string) barHint {
	h := hint(glyph, desc)
	h.key = glyph

	return h
}
