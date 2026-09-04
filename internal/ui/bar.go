package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
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
	target Target
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
	pip, state := pipOff, Chrome()
	if m.autopilotOn() {
		pip, state = pipOn, Paint(Live)
	}

	chips = append(chips, barChip{
		text: Chrome().Render("⚡ "+p.T("header.autopilot", "autopilot")) + " " + state.Render(pip) + " " + Paint(Live).Bold(true).Render("["+m.keys.Autopilot.Help().Key+"]"),
		// The switch, through the same target the header's own chip uses.
		target: Target{Kind: TargetStatusField, Field: "autopilot"},
	})

	// Interactive CLI chip
	if m.screen == screenList {
		chips = append(chips, barChip{
			text:   Chrome().Render("💬 "+p.T("header.cli_chip", "cli")) + " " + Paint(Live).Bold(true).Render("[c]"),
			target: Target{Kind: TargetBarHint, Key: "c"},
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
		chips = append(chips, barChip{text: Paint(Dim).Render(mark)})
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
	tail := Chrome().Render("[" + m.keys.Help.Help().Key + "] [" + m.keys.Quit.Help().Key + "]")
	chips := m.barFooterChips()
	chipsText := chipLine(chips)
	chipsW := lipgloss.Width(chipsText)
	hints := m.hints()

	for {
		leftStr := " " + strings.Join(append(drawn(hints), tail), hintGap)

		leftW := lipgloss.Width(leftStr)
		if leftW+chipsW+4 <= w && chipsText != "" {
			space := w - leftW - chipsW

			return leftStr + strings.Repeat(" ", space) + chipsText,
				place(hints), placeChips(chips, leftW+space)
		}

		if len(hints) == 0 {
			// No room for the chips, so they are not drawn and nothing at
			// that end of the bar is clickable.
			if leftW <= w {
				return fit(leftStr, w), place(hints), nil
			}

			return fit(leftStr, w), nil, nil
		}

		hints = hints[:len(hints)-1]
	}
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
func place(hints []barHint) []placedHint {
	out := make([]placedHint, 0, len(hints))
	x := 1

	for _, h := range hints {
		cells := lipgloss.Width(h.text)
		out = append(out, placedHint{key: h.key, x: x, w: cells})
		x += cells + lipgloss.Width(hintGap)
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
