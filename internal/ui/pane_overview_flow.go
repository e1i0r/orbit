package ui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// finishedPhases is every phase of this task that stopped, however it
// stopped.
func (m Model) finishedPhases() []view.Entry {
	var done []view.Entry

	for _, e := range m.entries {
		if e.Phase == "" {
			continue
		}

		switch e.What() {
		case view.EntryFinished, view.EntryFailed, view.EntryCancelled:
			done = append(done, e)
		}
	}

	return done
}

// overviewPhases is the record of the run: one row per phase with its verdict
// and its cost, and under each one what the model wrote, set as prose.
func (m Model) overviewPhases(t view.Task, w int) []string {
	p := m.opts.Words
	head := m.sectionHead(foldPhases, p.T("overview.execution_summary", "flows"),
		strconv.Itoa(len(m.finishedPhases())), w)

	if m.folded(foldPhases) {
		return []string{head, ""}
	}

	out := []string{head}

	if t.Band == view.Running {
		out = append(out, m.livePhase(t, w)...)
	}

	done := m.finishedPhases()
	if len(done) == 0 {
		if t.Band != view.Running {
			said := p.T("overview.no_phases", "no flow outputs recorded")
			if t.Band == view.ToDo {
				said = p.T("overview.not_started", "task has not been started yet (press [n] to start)")
			}

			out = append(out, paneGutter+Text(Tertiary).Render(said))
		}

		return append(out, "")
	}

	for _, ph := range done {
		out = append(out, paneGutter+m.phaseRow(ph))
		out = append(out, m.phaseSaid(ph, w)...)
		out = append(out, "")
	}

	return out
}

// phaseRow is the phase's name and its verdict on one line: the mark and the
// name carry the outcome, everything that qualifies it goes dim behind.
func (m Model) phaseRow(ph view.Entry) string {
	p := m.opts.Words
	mark, role := Paint(OK).Render("✓"), OK

	switch ph.What() {
	case view.EntryFailed:
		mark, role = Paint(Bad).Render("✗"), Bad
	case view.EntryCancelled:
		mark, role = Paint(Warn).Render("⏹"), Warn
	}

	row := mark + " " + Text(Primary).Bold(true).Render(ph.Phase)

	var qualifiers []string
	if ph.Cost > 0 {
		qualifiers = append(qualifiers, Text(Secondary).Render(fmt.Sprintf("$%.2f", ph.Cost)))
	}

	if ph.Gate != "" {
		qualifiers = append(qualifiers,
			Text(Secondary).Render(p.T("overview.gate", "gate {name}", about("name", ph.Gate))))
	}

	if ph.Cause != "" {
		qualifiers = append(qualifiers, Paint(role).Render(ph.Cause))
	}

	if len(qualifiers) == 0 {
		return row
	}

	return row + "  " + meta(qualifiers...)
}

// phaseSaid sets what the model wrote under its phase. Closed, a phase shows
// its opening paragraph, which is where an engine states what it did; open —
// [e] — it shows all of it. Either way it is wrapped at the measure and ruled
// down the left, so prose never arrives as a wall.
func (m Model) phaseSaid(ph view.Entry, w int) []string {
	text := strings.TrimSpace(ph.Said())
	if text == "" {
		return nil
	}

	if !m.expandedDetail {
		if cut := strings.Index(text, "\n\n"); cut > 0 {
			text = text[:cut]
		}

		if lines := prose(text, w, paneGutter+"  "); len(lines) > overviewFoldLines {
			return append(lines[:overviewFoldLines],
				paneGutter+"  "+Text(Tertiary).Render(
					proseRule+m.opts.Words.T("overview.more", "… [e] for all of it")))
		}
	}

	return prose(text, w, paneGutter+"  ")
}

// overviewFoldLines is how much of a phase's prose a closed row shows: enough
// to tell one phase from another, short enough that ten phases still fit on a
// screen.
const overviewFoldLines = 3

// livePhase is what the model is doing right now, drawn where the finished
// phases will be so the eye does not have to move when it lands.
func (m Model) livePhase(t view.Task, w int) []string {
	p := m.opts.Words
	now := orDef(t.CurrentAction, p.T("overview.running_model", "running model..."))
	glyph := m.runGlyph(working(t))

	// Cut to the pane and not to a constant. What the band has room for is
	// fifty characters, because five other fields share its row; this line
	// shares its row with nothing, and a command that fits in the window is
	// a command the reader gets to read.
	now = fit(now, max(20, w-lipgloss.Width(paneGutter)-lipgloss.Width(glyph)))

	out := []string{paneGutter + Paint(Live).Render(glyph) + Text(Primary).Bold(true).Render(now)}

	if t.CurrentThought != "" {
		out = append(out, prose(t.CurrentThought, w, paneGutter+"  ")...)
	}

	if t.ToolCallCount > 0 {
		out = append(out, paneGutter+"  "+Text(Secondary).Render(p.P("overview.tools", t.ToolCallCount,
			"{n} tool call", "{n} tool calls")))
	}

	return append(out, "")
}
