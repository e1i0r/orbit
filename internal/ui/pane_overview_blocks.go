package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
)

// overviewVitals is the dashboard row: what the task spent, how long it has
// been in this state, how many phases finished, and what it did to the tree.
// Four figures a reader compares at a glance, which is why they are drawn as
// cells rather than as four more rows of grey label and bold value.
//
// Under the cells go the dials, each next to the key that turns it, and the
// checkout the task runs in.
func (m Model) overviewVitals(t view.Task, w int) []string {
	p := m.opts.Words
	sum := parseDiffSummary(m.diff)

	cost := "—"
	if t.Cost > 0 {
		cost = fmt.Sprintf("$%.2f", t.Cost)
	}

	changed := plusMinus(sum, false)

	out := statStrip([]stat{
		{label: p.T("overview.cost", "cost"), value: cost, role: OK},
		{label: p.T("overview.duration", "duration"), value: elapsed(m.now, t.Since), role: Accent},
		{label: p.T("overview.phases", "phases"), value: strconv.Itoa(len(m.finishedPhases())), role: Accent},
		{label: p.T("overview.changed", "changed"), value: changed, role: Live},
	}, w)

	out = append(out, "")

	for _, l := range fields(m.dials(t), gridColumns(w), w-2*len(paneGutter)) {
		out = append(out, paneGutter+l)
	}

	if t.RepoPath != "" {
		out = append(out, paneGutter+Text(Tertiary).Render(tailFit(homeTilde(t.RepoPath), min(proseMeasure, w-2*len(paneGutter)))))
	}

	return append(out, "")
}

// dials is what the task would run on and the key that changes each one.
// The key sits against its own value rather than in the footer: [E] is not a
// gesture the pane offers, it is what edits the word in front of it.
func (m Model) dials(t view.Task) []field {
	// A task that has run carries its own engine and model. One that has not
	// shows what it would run on, which is the knob and then the setting
	// behind it — not the words claude and sonnet, which were the answer here
	// on builds that have neither.
	eng := orDef(t.Engine, m.dialEngine(m.knobs.Engine))

	models, _ := m.modelsFor(eng)
	mod := orDef(t.Model, orDef(m.knobs.Model, first(models)))

	// A window whose engines port answers nothing has no engine and no model
	// to name here, and a dash says so without naming one it has not.
	eng, mod = orDef(eng, unsetDial), orDef(mod, unsetDial)

	p := m.opts.Words
	dial := func(label, key, value string) field {
		return field{
			label: label,
			key:   key,
			value: Paint(Accent).Render(value),
		}
	}

	return []field{
		dial(p.T("overview.engine", "engine"), "k", eng+" "+mod),
		dial(p.T("overview.effort", "effort"), "E", orDef(m.knobs.Effort, "high")),
		dial(p.T("overview.thinking", "thinking"), "t", orDef(m.knobs.Thinking, "adaptive")),
		dial(p.T("overview.flow", "flow"), "F", orDef(t.Flow, flow.Default)),
	}
}

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
	head := m.sectionHead(foldPhases, p.T("overview.execution_summary", "phases"),
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
			said := p.T("overview.no_phases", "no phase outputs recorded")
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
		qualifiers = append(qualifiers, Text(Secondary).Render(p.T("overview.gate", "gate {name}", about("name", ph.Gate))))
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
				paneGutter+"  "+Text(Tertiary).Render(proseRule+m.opts.Words.T("overview.more", "… [e] for all of it")))
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

// overviewChanges is what the run did to the working tree: the two numbers
// that say how big it was, and the files it touched.
func (m Model) overviewChanges(w int) []string {
	p := m.opts.Words
	sum := parseDiffSummary(m.diff)
	head := m.sectionHead(foldChanges, p.T("overview.code_impact", "changes"), plusMinus(sum, false), w)

	if m.folded(foldChanges) {
		return []string{head, ""}
	}

	out := []string{head}

	if len(sum.files) == 0 && sum.added == 0 && sum.deleted == 0 {
		return append(out, paneGutter+Text(Tertiary).Render(p.T("overview.no_diff", "no working tree modifications recorded")), "")
	}

	out = append(out, paneGutter+meta(
		plusMinus(sum, true),
		Text(Secondary).Render(p.P("overview.files", len(sum.files), "{n} file", "{n} files")),
	))

	for i, f := range sum.files {
		if i >= overviewFileCap {
			rest := len(sum.files) - overviewFileCap

			return append(out, paneGutter+"  "+Text(Secondary).Render(p.P("overview.more_files", rest,
				"… and {n} more file", "… and {n} more files")), "")
		}

		out = append(out, paneGutter+"  "+Paint(OK).Render(fit(f, min(proseMeasure, w-2*len(paneGutter)-2))))
	}

	return append(out, "")
}

// gridColumns is how many columns the pane's two grids are laid in at a
// width. A cell has to hold the widest label with a space after it —
// THINKING [t] is twelve cells and MORE TESTS is ten — so under sixty the
// four are paired instead of run together. It is the width statStrip gives
// up its boxes at, and for the same reason.
func gridColumns(w int) int {
	if w < 60 {
		return 2
	}

	return 4
}

// overviewFileCap is how many paths the pane names before counting the rest.
const overviewFileCap = 4

// overviewActions is the toolbar: what this task can be moved along with,
// each key against the thing it does.
func (m Model) overviewActions(w int) []string {
	p := m.opts.Words

	act := func(key, label string) field {
		return field{label: label, value: Paint(Live).Render(key)}
	}

	head := m.sectionHead(foldDeliver, p.T("overview.quick_actions", "deliver"), "", w)
	if m.folded(foldDeliver) {
		return []string{head, ""}
	}

	out := []string{head}
	for _, l := range fields([]field{
		act("p", p.T("overview.action_pr", "create PR")),
		act("u", p.T("overview.action_update_pr", "update PR")),
		act("M", p.T("overview.action_merge_pr", "merge PR")),
		act("X", p.T("overview.action_close_pr", "close PR")),
		act("C", p.T("overview.action_checks", "fix checks")),
		act("T", p.T("overview.action_tests", "more tests")),
		act("R", p.T("overview.action_resolve", "resolve comments")),
		act("a", p.T("overview.action_feedback", "feedback")),
		act("0", p.T("overview.action_diff", "diff")),
	}, gridColumns(w), w-2*len(paneGutter)) {
		out = append(out, paneGutter+l)
	}

	return append(out, "")
}

// plusMinus is the two numbers a diff is judged by. A side that did not
// happen is left out rather than written as zero: "+21" is a fact, and
// "+21 −0" sends the reader to check a number that was never in question.
//
// The cells of the strip paint their own value, so painted is false there:
// a colour inside a cell would end mid-word where lipgloss resets it.
func plusMinus(sum diffSummary, painted bool) string {
	paint := func(r Role, s string) string {
		if painted {
			return Paint(r).Render(s)
		}

		return s
	}

	var parts []string

	if sum.added > 0 {
		parts = append(parts, paint(OK, fmt.Sprintf("+%d", sum.added)))
	}

	if sum.deleted > 0 {
		parts = append(parts, paint(Bad, fmt.Sprintf("−%d", sum.deleted)))
	}

	if len(parts) == 0 {
		return "—"
	}

	return strings.Join(parts, " ")
}

// homeTilde writes the reader's home directory as ~. Which checkout this is
// is the useful half of the path; whose machine it sits on they know.
func homeTilde(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || !strings.HasPrefix(path, home) {
		return path
	}

	return "~" + strings.TrimPrefix(path, home)
}

// tailFit keeps the end of a path when the whole of it will not fit: the last
// segments say which checkout this is, the first ones say whose machine it is.
func tailFit(s string, w int) string {
	if w <= 1 || lipgloss.Width(s) <= w {
		return s
	}

	r := []rune(s)

	return "…" + string(r[len(r)-(w-1):])
}
