package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// barHint is one entry of the key bar: the hint as it is drawn, and the
// keystroke it stands for.
type barHint struct {
	key  string
	text string
}

// placedHint is one hint of the drawn bar and the cells it occupies, counted
// from the left edge of the terminal.
type placedHint struct {
	key  string
	x, w int
}

// barChip is one of the badges at the right end of the bar: what is drawn,
// and what a click on it means. Where it ends up is barLayout's answer,
// because barLayout is the only thing that knows where the right end is —
// the widths are of translated words and vary with the language.
type barChip struct {
	text   string
	target point.Target
}

// chipGap is what the chips are joined with, and what placeChips steps over.
const chipGap = "    "

// barLine is what can be pressed right now.
func (m Model) barLine(w int) string {
	line, _, _ := m.barLayout(w)
	return line
}

// barFooterChips renders autopilot and interactive CLI on the footer right side.
func (m Model) barFooterChips() []barChip {
	p := m.opts.Words

	var chips []barChip

	// Autopilot chip. The label is the bar's own ink like every other label
	// on this line, and the only thing that carries a colour is the pip,
	// which is the one part of the chip that is saying something.
	pip, state := pipOff, theme.Chrome()
	if m.autopilotOn() {
		pip, state = pipOn, theme.Paint(theme.Live)
	}

	chips = append(chips, barChip{
		text: theme.Chrome().Render("⚡ "+p.T("header.autopilot", "autopilot")) + " " + state.Render(pip) + " " + theme.Paint(theme.Live).Bold(true).Render("["+m.keys.Autopilot.Help().Key+"]"),
		// The switch, through the same target the header's own chip uses.
		target: point.Target{Kind: point.StatusField, Field: "autopilot"},
	})

	// Interactive CLI chip
	if m.screen == screenList {
		chips = append(chips, barChip{
			text:   theme.Chrome().Render("💬 "+p.T("header.cli_chip", "cli")) + " " + theme.Paint(theme.Live).Bold(true).Render("[c]"),
			target: point.Target{Kind: point.BarHint, Key: "c"},
		})
	}

	// Version chip, at the far end of the bar.
	//
	// It is read and not pressed — there is no screen behind a version — so
	// it carries no target, and a click on it lands on nothing the way a
	// click on the empty end of the bar does. It is last because it is the
	// one thing here that never changes while the window is open: the eye
	// goes looking for it, rather than being caught by it.
	//
	// A build with nothing stamped in it says nothing. That is `go test`,
	// where a version would be a fact about the harness rather than about
	// orbit.
	if mark := buildMark(m.opts.Version); mark != "" {
		// The bar's own ink, like every other label on this line: Dim is
		// faint grey on the bar's grey, which is a version that is there and
		// cannot be read.
		chips = append(chips, barChip{text: theme.Chrome().Render(mark)})
	}

	return chips
}

// chipLine is the chips as barLayout draws them.
func chipLine(chips []barChip) string {
	out := make([]string, 0, len(chips))
	for _, c := range chips {
		out = append(out, c.text)
	}

	return strings.Join(out, chipGap)
}

// placeChips walks the chips the way chipLine joined them and says where
// each one starts, measuring in cells from the left edge of the terminal.
//
// It is here rather than in hitBar for the reason place is: a click is
// answered by where the thing was drawn, not by a width somebody guessed.
// The guess was two constants — the last 28 columns were the cli chip and
// the 32 before it the switch — which was wrong in Spanish, where both
// words are longer, and wrong at every width where barLayout drops the
// chips altogether: clicking the empty end of the bar opened a shell.
func placeChips(chips []barChip, x int) []headerZone {
	out := make([]headerZone, 0, len(chips))

	for _, c := range chips {
		cells := lipgloss.Width(c.text)
		out = append(out, headerZone{target: c.target, x: x, w: cells})
		x += cells + lipgloss.Width(chipGap)
	}

	return out
}

// barLayout is the key bar, drawn, and where it put each hint.
func (m Model) barLayout(w int) (string, []placedHint, []headerZone) {
	// The three keys that are always there, in the corner they have always
	// been in: the menu of everything the thing under the cursor can be
	// asked, the cheat sheet, and the way out.
	//
	// Hints of their own rather than one string, so that a click on them is
	// answered. Drawn and not placed, they were three keys the bar promises
	// and only the keyboard could reach — which is the one thing the window
	// says it never does. The way out keeps no key of its own: a window that
	// closed on a stray click in the corner is a window that lost whatever
	// the reader was reading, and the q it draws is the way to mean it.
	tailHints := []barHint{
		{key: m.keys.Menu.Help().Key, text: theme.Chrome().Render("[" + m.keys.Menu.Help().Key + "]")},
		{key: m.keys.Help.Help().Key, text: theme.Chrome().Render("[" + m.keys.Help.Help().Key + "]")},
		{text: theme.Chrome().Render("[" + m.keys.Quit.Help().Key + "]")},
	}
	// The board's menu is offered where it is a different door from m:
	// on a task's own screen, and on the list when the cursor is on a
	// row. With no row under the cursor m already opens the board's menu,
	// and two hints for one answer is a bar making a distinction the
	// window does not. A dialog or a form takes every keystroke while it
	// is up, so neither offers it at all.
	if m.boardMenuDiffers() {
		tailHints = append([]barHint{tailHints[0], {
			key:  m.keys.Board.Help().Key,
			text: theme.Chrome().Render("[" + m.keys.Board.Help().Key + "]"),
		}}, tailHints[1:]...)
	}

	// A screen of its own answers none of the three: [m] and [?] did
	// nothing on settings, and on the supervisor they were typed into the
	// line. Its own keys are in its footer.
	if !m.onBoardKeys() {
		tailHints = nil
	}

	corner := strings.Join(drawn(tailHints), " ")
	all := m.barFooterChips()
	hints := m.hints()

	// What gives way first, and in what order.
	//
	// The chips were fixed and the hints gave way to them, which in
	// Spanish at a hundred columns left a bar reading `[m] [M] [?] [q]`
	// and nothing else: no move, no open, and no `n`, under a band saying
	// "pulsa n para poner una en marcha". Sixty-eight of those cells were
	// three chips, and the last of them is a version number.
	//
	// Making the chips give way first was the other mistake. At a hundred
	// and twenty columns there is room for all of it, and a bar that
	// spends every spare cell on a seventh affordance rather than on the
	// version is a bar nobody can quote in a bug report.
	//
	// So the spare verbs go first, then the chips, and the first few
	// verbs are protected from both. The autopilot switch is last of all,
	// because it is the only thing on this line that is a control with a
	// state.
	floor := min(essentialHints, len(hints))

	// 1. The affordances past the first few, while every chip stays.
	for keep := len(hints); keep >= floor; keep-- {
		if line, at, zones, ok := m.fitBar(w, hints[:keep], tailHints, corner, all); ok {
			return line, at, zones
		}
	}

	// 2. The chips, down to the switch, while the first few verbs stay.
	for keep := len(all); keep >= switchOnly; keep-- {
		if line, at, zones, ok := m.fitBar(w, hints[:floor], tailHints, corner, all[:keep]); ok {
			return line, at, zones
		}
	}

	// 3. Those verbs too, beside the switch on its own.
	for keep := floor; keep >= 0; keep-- {
		if line, at, zones, ok := m.fitBar(w, hints[:keep], tailHints, corner, all[:switchOnly]); ok {
			return line, at, zones
		}
	}

	// 4. Not even the corner beside the switch. Nothing at that end of
	// the bar is drawn, and nothing there is clickable.
	leftStr := " " + corner
	if lipgloss.Width(leftStr) <= w {
		return cells.Fit(leftStr, w), place(nil, tailHints), nil
	}

	return cells.Fit(leftStr, w), nil, nil
}

// essentialHints is how many of the bar's verbs are worth more than any
// chip: where the cursor goes, how to open what it is on, and the one
// thing to do with it. Below that the bar stops answering the band, which
// says things like "pulsa n para poner una en marcha".
const essentialHints = 3

// switchOnly is the chips cut down to the autopilot switch, which is the
// last of them to go.
const switchOnly = 1

// fitBar lays the line out with one set of hints and one set of chips, and
// says whether it fitted. Every rank of barLayout's search asks it the same
// question with a smaller set.
func (m Model) fitBar(
	w int, hints, tailHints []barHint, corner string, chips []barChip,
) (string, []placedHint, []headerZone, bool) {
	text := chipLine(chips)
	leftStr := " " + strings.Join(append(drawn(hints), corner), hintGap)
	leftW := lipgloss.Width(leftStr)

	if text == "" || leftW+lipgloss.Width(text)+4 > w {
		return "", nil, nil, false
	}

	space := w - leftW - lipgloss.Width(text)

	return leftStr + strings.Repeat(" ", space) + text,
		place(hints, tailHints), placeChips(chips, leftW+space), true
}

// drawn is the hints as barLine joins them.
func drawn(hints []barHint) []string {
	out := make([]string, 0, len(hints)+1)
	for _, h := range hints {
		out = append(out, h.text)
	}

	return out
}

// place walks the hints the way the line was joined and says where each one
// starts, measuring in cells.
func place(hints, tail []barHint) []placedHint {
	out := make([]placedHint, 0, len(hints)+len(tail))
	x := 1

	for _, h := range hints {
		cells := lipgloss.Width(h.text)
		out = append(out, placedHint{key: h.key, x: x, w: cells})
		x += cells + lipgloss.Width(hintGap)
	}

	// The three in the corner are joined to each other by one space and to
	// the hints before them by hintGap, which is what barLayout draws — a
	// zone stepped along by the wrong gap is a zone over the key beside the
	// one it names, and that is the whole failure this is here to avoid.
	for _, h := range tail {
		cells := lipgloss.Width(h.text)
		out = append(out, placedHint{key: h.key, x: x, w: cells})
		x += cells + 1
	}

	return out
}

// buildMark is the running build's version, cut down to what a corner of the
// bar has room for.
//
// A release is stamped with its tag and needs nothing done to it. A build
// from a checkout is stamped by `git describe`, which spells out how far past
// the tag it is and which commit that was — "v0.1.83-27-gabd558b-dirty" is
// twenty-five cells to say three things, and only two of them are worth the
// room: the release this came after, and that it is not that release. So the
// commit goes and what is left is v0.1.83+27, with a star when the tree the
// build was made from had uncommitted work in it.
func buildMark(version string) string {
	v := strings.TrimSpace(version)
	if v == "" || v == "dev" {
		return v
	}

	star := ""
	if cut := strings.TrimSuffix(v, "-dirty"); cut != v {
		v, star = cut, "*"
	}

	// A checkout with no tags describes as a bare commit, which is not a
	// version and does not take a v.
	if v != "" && (v[0] == 'v' || (v[0] >= '0' && v[0] <= '9')) {
		v = "v" + strings.TrimPrefix(v, "v")
	}

	// The tail git describe adds: -<commits since the tag>-g<commit>.
	if at := strings.LastIndex(v, "-g"); at > 0 {
		if cut := strings.LastIndex(v[:at], "-"); cut > 0 {
			v = v[:cut] + "+" + v[cut+1:at]
		}
	}

	return v + star
}
