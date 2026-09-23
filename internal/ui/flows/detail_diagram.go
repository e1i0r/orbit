package flows

// The pipeline drawn as boxes and arrows, on the screen that shows what a
// flow does.
//
// Its own file because detail_view.go went over the ceiling with both in
// it, and a diagram and a reading of a flow are two subjects: this one is
// geometry, and nothing in it knows what a phase is for.

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// What a phase's box is drawn at: the floor a short name is held to, the
// margin the stacked form is drawn in, and the narrowest box that is still
// a box, for a window with no room for anything.
const (
	diagramBoxFloor = 14
	diagramIndent   = 2
	diagramBoxMin   = 4
)

// renderFlowDiagram builds an ASCII box-and-arrow flowchart diagram for the given phases.
func renderFlowDiagram(phases []flow.Phase, maxW int) []string {
	if len(phases) == 0 {
		return nil
	}

	type box struct {
		top, mid1, mid2, bot string
		width                int
	}

	var boxes []box

	for i, ph := range phases {
		line1 := fmt.Sprintf("%d. %s", i+1, ph.Name)

		// A loop names no engine of its own — what runs is the phase inside
		// it — so the box says how many turns it takes, where a phase says
		// what runs it. It used to say "/def": an empty engine and a model
		// nobody set, which reads as a phase that was never configured.
		line2 := fmt.Sprintf("%s/%s", ph.Engine, cells.OrDef(ph.Model, "def"))
		if ph.Loop != nil {
			line2 = "↻ ×" + strconv.Itoa(ph.Loop.Max)
		}

		if runsIn(ph).FeedOutput {
			line2 += " ➔"
		}

		if ph.Wait {
			line2 += " ⏸"
		}

		// As wide as its widest line and the two sides it is drawn in,
		// never under the floor that keeps a one-word phase from reading
		// as a tick box — and never wider than the room the diagram was
		// given.
		//
		// That last was missing. The stack this falls back to when the
		// boxes will not sit side by side is drawn two cells in and was
		// never measured against anything, so a phase with a long name
		// drew a box ten cells past the right of the window: the
		// terminal wraps it, and every row of the reading below it lands
		// where the reader is not looking.
		boxW := min(
			max(max(lipgloss.Width(line1), lipgloss.Width(line2))+2, diagramBoxFloor),
			max(maxW-diagramIndent, diagramBoxMin),
		)

		padLine1 := cells.Pad(line1, boxW-2, false)
		padLine2 := cells.Pad(line2, boxW-2, false)

		b := box{
			top:   "┌" + strings.Repeat("─", boxW-2) + "┐",
			mid1:  "│" + padLine1 + "│",
			mid2:  "│" + padLine2 + "│",
			bot:   "└" + strings.Repeat("─", boxW-2) + "┘",
			width: boxW,
		}
		boxes = append(boxes, b)
	}

	arrow := " ──▶ "
	arrowPad := "     "

	var rowTop, rowMid1, rowMid2, rowBot string

	for i, b := range boxes {
		if i > 0 {
			rowTop += arrowPad
			rowMid1 += arrow
			rowMid2 += arrowPad
			rowBot += arrowPad
		}

		rowTop += b.top
		rowMid1 += b.mid1
		rowMid2 += b.mid2
		rowBot += b.bot
	}

	if lipgloss.Width(rowTop) <= maxW {
		return []string{
			theme.Paint(theme.Dim).Render(rowTop),
			theme.Paint(theme.Accent).Render(rowMid1),
			theme.Paint(theme.OK).Render(rowMid2),
			theme.Paint(theme.Dim).Render(rowBot),
		}
	}

	// Fallback to vertical stack when horizontal space is limited
	var out []string

	for i, b := range boxes {
		if i > 0 {
			out = append(out, "        │", "        ▼")
		}

		out = append(out,
			theme.Paint(theme.Dim).Render("  "+b.top),
			theme.Paint(theme.Accent).Render("  "+b.mid1),
			theme.Paint(theme.OK).Render("  "+b.mid2),
			theme.Paint(theme.Dim).Render("  "+b.bot),
		)
	}

	return out
}

// phaseCards is one card per phase: what runs it, how it is joined to the
// phase before, and what it is told to do.
//
// It is shared by the flow inspector and the designer's diagram tab, because
// they are two windows onto the same list of phases and a second copy of
// this would be a second answer to "what does this flow do".
func (s State) phaseCards(phases []flow.Phase, w int, e Env) []string {
	var out []string

	for i, ph := range phases {
		out = append(out, s.phaseCard(i, ph, w, e)...)
	}

	return out
}
