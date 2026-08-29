package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

type gateCheck struct {
	name     string
	duration string
	passed   bool
	command  string
	reason   string
	phase    string
}

// gateNameCells is the gates tab's name column, and gateWordCells the verdict
// beside it. Sixteen fits the gate names a flow declares and truncates a
// longer one, which is the trade the log's phase column already makes.
const (
	gateNameCells = 16
	gateWordCells = 6
)

// gatesLines is the gates tab's content, ready for the pane.
func (m Model) gatesLines() []string {
	lines, _ := m.gatesRows()

	return lines
}

// gatesRows is that content and, beside it, which check each row that folds
// stands for. The two are built in one pass for the reason logRows is: a hit
// test that counted the rows again would be a second opinion about where a
// row is.
func (m Model) gatesRows() ([]string, map[int]int) {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.errSaid(m.logErr))}, nil
	}

	var checks []gateCheck

	for _, e := range m.entries {
		if e.What() == view.EntryGatePassed || e.What() == view.EntryGateFailed {
			gName := e.Gate
			if gName == "" {
				gName = "check"
			}

			cmd := e.Text
			if cmd == "" {
				cmd = e.Tool
			}

			checks = append(checks, gateCheck{
				name:     gName,
				passed:   e.What() == view.EntryGatePassed,
				command:  cmd,
				reason:   e.Cause,
				phase:    e.Phase,
				duration: "ok",
			})
		}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("gates.title", "Verification Gates & Checks")),
		"  " + Paint(Dim).Render(p.T("gates.subtitle", "what needs to pass — by attempt, and why it stopped")),
		"",
	}

	if len(checks) == 0 {
		return append(out, "  "+Paint(Dim).Render(p.T("gates.empty", "no verification gates have run for this task"))), nil
	}

	passedCount := 0

	for _, c := range checks {
		if c.passed {
			passedCount++
		}
	}

	summaryRole := OK
	summaryWord := p.T("gates.pass", "pass")

	if failed := len(checks) - passedCount; failed > 0 {
		summaryRole = Bad
		summaryWord = p.P("gates.badge_failed", failed, "{n} failed", "{n} failed")
	}

	out = append(out, fmt.Sprintf("  %s %s   %d/%d %s   %s",
		Paint(Accent).Render("▼"),
		Paint(Accent).Bold(true).Render(p.T("gates.attempt", "attempt 1")),
		passedCount, len(checks),
		p.T("gates.passed_word", "passed"),
		Paint(summaryRole).Bold(true).Render(summaryWord),
	))
	out = append(out, "")

	heads, w := map[int]int{}, max(m.frame.Body.W, 1)

	for i, c := range checks {
		rows, folds := m.gateRows(c, i, w)
		if folds {
			heads[len(out)] = i
		}

		out = append(out, rows...)
	}

	return append(out, ""), heads
}

// gateRows is one check — whether it passed, what it ran, and why it stopped
// — and whether there is more to it than the row is showing.
//
// Whether it folds is decided here, by wrapping what it has to the measure it
// will be drawn at: a check whose whole story fits beside its name is offered
// no arrow, because opening it would put nothing new on the screen.
func (m Model) gateRows(c gateCheck, i, w int) ([]string, bool) {
	p := m.opts.Words

	icon, word, role := "✅", p.T("gates.pass", "pass"), OK
	if !c.passed {
		icon, word, role = "❌", p.T("gates.fail", "fail"), Bad
	}

	// The columns are padded on the plain word and painted afterwards: a
	// width verb counts the bytes of an escape sequence as cells, so a
	// padded rendered string is a column that moves with the palette.
	head := "  " + Paint(role).Render(icon) + " " +
		Paint(Accent).Render(pad(c.name, gateNameCells, false)) + "  " +
		Paint(role).Render(pad(word, gateWordCells, false)) + "  "

	lead := 2 + lipgloss.Width(icon) + 1 + gateNameCells + 2 + gateWordCells + 2
	availW := max(20, w-lead-lipgloss.Width(foldShut)-2)

	said := []string{c.command}
	if !c.passed && c.reason != "" {
		said = append(said, p.T("gates.why_failed", "why it failed ·")+" "+c.reason)
	}

	// Wrapped and then cut: a gate is a shell command, and a command with a
	// long path in it has nothing to break at, so a row can come back from
	// the wrap longer than it was wrapped to.
	var body []string

	for _, l := range said {
		for _, wl := range splitIntoLines(l, availW) {
			body = append(body, fit(wl, availW))
		}
	}

	if len(body) == 0 {
		return []string{strings.TrimRight(head, " ")}, false
	}

	if len(body) == 1 {
		return []string{head + strings.Repeat(" ", lipgloss.Width(foldShut)) + Paint(Dim).Render(body[0])}, false
	}

	open := m.rowOpen(tabGates, i)
	mark := Text(Tertiary).Render(foldMark(open))

	if !open {
		return []string{head + mark + Paint(Dim).Render(body[0])}, true
	}

	out := []string{head + mark + Text(Secondary).Render(body[0])}

	indent := strings.Repeat(" ", lead+lipgloss.Width(foldShut))
	for _, l := range body[1:] {
		out = append(out, indent+Text(Secondary).Render(l))
	}

	return out, true
}
