package ui

// The start dialog, drawn. What it decides is in start.go; this is the four
// blocks it decides them with, in the order they are read: which task, which
// flow, what that flow is made of, and whether any of it will stop for a
// person.
//
// It is a file of its own for the reason detail.go and detailkeys.go are two
// files: the 300-line ceiling does not hold a screen's behaviour and its
// geometry in one, and the cut falls where a reader would put it anyway.

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

// startIndent is the three cells every line of the dialog is inset by, which
// is the gutter the board draws its cursor in. The two screens line up
// because one replaces the other in the same region.
const startIndent = "   "

// startGap is the space between two fields of a phase row. Three, so that a
// short model name and the column after it are not read as one word.
const startGap = 3

// startRows draws the dialog into the body region.
//
// The order the blocks are appended in is the order they are given up as the
// terminal shortens, because fill cuts from the end. That is the right
// order: a reader who cannot see the switch can still press A and read the
// band, and a reader who cannot see which task they are about to start has
// no other way to find that out.
func (m Model) startRows(h, w int) []string {
	if h <= 0 || w <= 0 {
		return nil
	}
	out := []string{m.startHead(w), "", m.flowLine(w), ""}
	out = append(out, m.phaseRows(w)...)
	out = append(out, "")
	return fill(append(out, m.autopilotRows(w)...), h)
}

// startHead names the task the run would be: its id and title on the left,
// the repository on the right.
//
// It is detailHead's shape on purpose. These are the two things one row can
// open into, and a heading that moved between them would be a heading nobody
// reads — which is how, in the program this replaces, a diff got applied to
// the wrong branch.
func (m Model) startHead(w int) string {
	t, ok := m.task(m.start.id)
	left := Paint(Accent).Render(m.start.id)
	if !ok {
		return spread(" "+left, Paint(Dim).Render(m.opts.Words.T("detail.gone",
			"this task is no longer on the board")), w)
	}
	if t.Title != "" {
		left += "  " + Paint(Dim).Render(t.Title)
	}
	return spread(" "+left, Paint(Dim).Render(t.Repo), w)
}

// flowLine is the first line and the one that changes the rest: the flow
// showing on the left, the order f visits them in on the right.
func (m Model) flowLine(w int) string {
	p := m.opts.Words
	chosen := m.start.chosen()
	left := startIndent + Paint(Dim).Render(p.T("start.flow", "flow")) + "  " +
		Paint(Accent).Render(chosen.name)
	if mark := m.flowMark(chosen); mark != "" {
		left += "  " + Paint(Dim).Render(mark)
	}
	cycle := m.start.cycle()
	if len(cycle) < 2 {
		return fit(left, w)
	}
	parts := make([]string, 0, len(cycle))
	for i, f := range cycle {
		role := Dim
		if i == len(cycle)-1 {
			role = Accent
		}
		parts = append(parts, Paint(role).Render(f.name))
	}
	return spread(left, strings.Join(parts, Paint(Dim).Render(dot)), w)
}

// flowMark says where a flow came from, in the words `orbit flows` uses.
//
// Word for word the same words, and not by anyone remembering to keep them
// so: both screens say a mark through the same catalogue key, and the
// translation honesty test fails if two call sites give one key two
// different English sentences. A reader who ran the command and then opened
// this dialog is looking at the same fact, and two spellings of it would
// read as two facts.
//
// A built-in is unmarked here and marked in the listing, and that difference
// is deliberate. The listing is a catalogue of everything there is, where the
// question is where each one came from; the dialog is one line about the flow
// that is about to run, where a mark on every line is not a mark.
//
// A name nothing answers to is unmarked too. The dialog says what is wrong
// with it in the error line below, which is a sentence and not a mark.
func (m Model) flowMark(f startFlow) string {
	p := m.opts.Words
	switch f.origin {
	case flow.OriginShadow:
		return p.T("flow.shadowing", "yours, shadowing the built-in")
	case flow.OriginUser:
		return p.T("flow.yours", "yours")
	}
	return ""
}

// phaseRows is what the flow on screen is made of, one phase to a line.
//
// The mark is on the first phase and it does not move, because there is
// nothing here to select: it says where the run begins, not where a cursor
// is. The day a phase can be held back or edited is the day it becomes a
// cursor, and start.go says why that day is not this one.
func (m Model) phaseRows(w int) []string {
	f := m.start.chosen()
	if f.err != nil {
		return []string{fit(startIndent+Paint(Bad).Render(f.err.Error()), w)}
	}
	var nameW, engineW, modelW int
	for _, ph := range f.flow.Phases {
		nameW = max(nameW, lipgloss.Width(ph.Name))
		engineW = max(engineW, lipgloss.Width(ph.Engine))
		modelW = max(modelW, lipgloss.Width(ph.Model))
	}
	out := make([]string, 0, len(f.flow.Phases))
	for i, ph := range f.flow.Phases {
		mark := " "
		if i == 0 {
			mark = markGlyph
		}
		fields := []struct {
			cells int
			text  string
		}{{nameW, ph.Name}, {engineW, ph.Engine}, {modelW, ph.Model}, {0, m.phaseNote(ph)}}
		var parts []string
		for _, field := range fields {
			if field.cells == 0 && field.text == "" {
				continue
			}
			parts = append(parts, Paint(Dim).Render(pad(field.text, max(field.cells, lipgloss.Width(field.text)), false)))
		}
		out = append(out, fit(startIndent+Paint(Accent).Render(mark)+" "+
			strings.Join(parts, strings.Repeat(" ", startGap)), w))
	}
	return out
}

// phaseNote is the one thing worth knowing about a phase before it runs:
// what it may touch, and whether it will stop when it is finished.
//
// Both are said, where the specified screen showed only one of them. A
// permission that disappears off the screen because the phase happens to
// wait is precisely the silent widening internal/flow's closed vocabulary
// exists to prevent — a reader deciding whether to spend an hour on this
// should not have to know that one fact hides the other.
func (m Model) phaseNote(ph flow.Phase) string {
	p := m.opts.Words
	note := p.T("start.asks_nothing", "asks for nothing")
	if len(ph.Permissions) > 0 {
		note = p.T("start.permissions", "permissions: {list}",
			about("list", strings.Join(ph.Permissions, ", ")))
	}
	if ph.Wait {
		note += dot + p.T("start.stops", "stops when it is done")
	}
	return note
}

// autopilotRows is the standing switch, both positions shown.
//
// Both, rather than the current one alone, because this is the moment the
// setting matters most and "autopilot ● on" answers a different question
// from "what happens if it is off". The pips are the header's own glyphs, so
// the switch a reader flips here is recognisably the switch they can see up
// there.
func (m Model) autopilotRows(w int) []string {
	p := m.opts.Words
	on := m.autopilotOn()
	rows := []struct {
		picked bool
		word   string
		what   string
	}{
		{on, p.T("start.on", "on"), p.T("start.autopilot_on", "every phase runs without asking")},
		{!on, p.T("start.off", "off"), p.T("start.autopilot_off", "stops at each phase")},
	}
	label := p.T("start.autopilot", "autopilot")
	lead := []string{
		startIndent + Paint(Dim).Render(label) + "  ",
		startIndent + strings.Repeat(" ", lipgloss.Width(label)+2),
	}
	var wordW int
	for _, r := range rows {
		wordW = max(wordW, lipgloss.Width(r.word))
	}
	out := make([]string, 0, len(rows))
	for i, r := range rows {
		pip, role := pipOff, Dim
		if r.picked {
			pip, role = pipOn, Live
		}
		out = append(out, fit(lead[i]+Paint(role).Render(pip+" "+pad(r.word, wordW, false))+
			strings.Repeat(" ", startGap)+Paint(Dim).Render(r.what), w))
	}
	return out
}
