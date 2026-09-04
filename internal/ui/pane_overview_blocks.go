package ui

import (
	"fmt"
	"os"
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
		{
			label: p.T("overview.phases", "flow"),
			value: orDef(t.Flow, flow.Default),
			role:  Accent,
		},
		{label: p.T("overview.changed", "changed"), value: changed, role: Live},
	}, w)

	out = append(out, "")

	for _, l := range fields(m.dials(t), gridColumns(w), w-2*len(paneGutter)) {
		out = append(out, paneGutter+l)
	}

	if t.RepoPath != "" {
		tail := tailFit(homeTilde(t.RepoPath), min(proseMeasure, w-2*len(paneGutter)))
		out = append(out, paneGutter+Text(Tertiary).Render(tail))
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

// overviewChanges is what the run did to the working tree: the two numbers
// that say how big it was, and the files it touched.
func (m Model) overviewChanges(w int) []string {
	p := m.opts.Words
	sum := parseDiffSummary(m.diff)
	head := m.sectionHead(foldChanges, p.T("overview.code_impact", "changes"),
		plusMinus(sum, false), w)

	if m.folded(foldChanges) {
		return []string{head, ""}
	}

	out := []string{head}

	if len(sum.files) == 0 && sum.added == 0 && sum.deleted == 0 {
		msg := p.T("overview.no_diff", "no working tree modifications recorded")
		return append(out, paneGutter+Text(Tertiary).Render(msg), "")
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

		trimmed := fit(f, min(proseMeasure, w-2*len(paneGutter)-2))
		out = append(out, paneGutter+"  "+Paint(OK).Render(trimmed))
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
		act("D", p.T("overview.action_review", "deep review")),
		act("a", p.T("overview.action_feedback", "feedback")),
		act("0", p.T("overview.action_diff", "diff")),
	}, gridColumns(w), w-2*len(paneGutter)) {
		out = append(out, paneGutter+l)
	}

	return append(append(out, m.handOutRows()...), "")
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
