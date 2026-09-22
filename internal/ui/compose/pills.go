package compose

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/prose"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

const composeLabelWidth = 14

func composeLabel(label string, active bool) string {
	mark := strings.Repeat(" ", cells.Gutter)
	if active {
		mark = cells.Mark + strings.Repeat(" ", cells.Gutter-1)
	}

	padded := cells.Pad(label+":", composeLabelWidth, false)

	return mark + theme.Paint(theme.Dim).Render(padded) + " "
}

func (s State) composeFlowLine(active bool, w int, e Env) string {
	p := e.Words
	prefix := composeLabel(p.T("compose.flow", "flow"), active)

	// The row is internal/ui/prose's, and the start dialog draws the same
	// one. Two drawings of one row is two rows that drift: the dialog's
	// was already a worse version of this, showing only the flow it was
	// on and cycling blind to the rest.
	choices := make([]prose.Choice, 0, len(s.flows))
	for _, f := range s.flows {
		choices = append(choices, prose.Choice{Label: f, Glyph: prose.FlowGlyph(f)})
	}

	// One line here, as this form has always drawn it: the width is
	// passed as nought, which Row reads as "do not wrap". The form's
	// caret arithmetic is per line and the box below it is placed from a
	// fixed plan, so a flow row that grew a second line would move
	// everything under it. The start dialog wraps because its plan is
	// built from what each block actually took.
	rows, _ := prose.Row(choices, s.flowIdx, 0, 0)

	newBtn := theme.Pill(" ➕ "+p.T("compose.new_flow_btn", "New")+" ", theme.PillInk, theme.PillCreate)

	line := prefix + rows[0] + " " + newBtn
	if active {
		line += " " + theme.Paint(theme.Dim).Render(p.T("compose.flow_hint", "(←/→ to cycle, click again/i for details, + new)"))
	}

	return cells.Fit(line, w)
}

// flowDetail is what the flow the form is set to will actually do: one row
// per phase, and what the flow says about itself under them.
//
// It was one row for all of it — every phase and the whole description run
// together behind a "·" — which at any terminal width ended in an ellipsis,
// so the sentence a reader was choosing between flows on was the half of it
// that fitted. A phase is a line because a flow is a sequence, and a
// sequence read as a paragraph is a sequence nobody counts.
//
// The description is wrapped to the room left beside the label and cut at
// three rows: it is prose somebody wrote about their own flow, and a form
// that grows without bound underneath the field being edited pushes the task
// box off a short terminal.
func (s State) flowDetail(name string, w int, e Env) []string {
	fl, err := flow.Resolve(e.Flows, name)
	if err != nil {
		return nil
	}

	out := make([]string, 0, len(fl.Phases)+flowAboutRows)

	for i, ph := range fl.Phases {
		out = append(out, s.flowPhaseRow(i+1, ph, e))
	}

	about := cells.WrapKeeping(fl.Description, max(w-composeLabelStart-2, minFlowAbout))
	if len(about) > flowAboutRows {
		about = append(about[:flowAboutRows-1], about[flowAboutRows-1]+"…")
	}

	return append(out, about...)
}

// flowAboutRows is how many rows of a flow's own description are shown, and
// minFlowAbout the narrowest the wrap is allowed to get — a pane too narrow
// to hold the label and a word would otherwise wrap one letter per row.
const (
	flowAboutRows = 3
	minFlowAbout  = 20
)

// flowPhaseRow is one phase: what it is called, what runs it, and whether it
// stops there.
//
// The pause is a word and not only the glyph it used to be. A flow that
// stops halfway and waits is the one thing about a flow that surprises the
// person who chose it, and ⏸ alone is a symbol a reader has to already know.
func (s State) flowPhaseRow(n int, ph flow.Phase, e Env) string {
	row := fmt.Sprintf("%d %s", n, ph.Name)

	if ph.Engine != "" {
		engine := ph.Engine
		if ph.Model != "" && ph.Model != "default" {
			engine += "/" + ph.Model
		}

		row += " · " + engine
	}

	if ph.Wait {
		return row + " · " + theme.Paint(theme.Warn).Render("⏸ "+e.Words.T("compose.flow_waits", "waits for you"))
	}

	return row
}

const composeLabelStart = cells.Gutter + composeLabelWidth + 1

func composePillWidth(name string, selected bool) int {
	if selected {
		return lipgloss.Width(theme.Pill(" ● "+name+" ", theme.PillInkLit, theme.PillInk))
	}

	return lipgloss.Width(theme.Pill(" "+name+" ", theme.PillInkRest, theme.PillRest))
}
