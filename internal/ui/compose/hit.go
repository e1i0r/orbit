package compose

// Where a cell of the compose form is: the fields a task is written into and
// the rows of pills each one is chosen from.
//
// It is its own file because the form is its own screen — three fields and
// the row of flows they are written against — and reading the task view's
// hit test should not mean reading past all of it.

import (
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

func (s State) hitComposeFlowPills(x int, field int, e Env) point.Target {
	p := e.Words
	curX := composeLabelStart

	for i, f := range s.flows {
		glyph := "⚡ "

		switch f {
		case "quick":
			glyph = "🚀 "
		case "careful":
			glyph = "🛡️ "
		}

		pillWidth := composePillWidth(glyph+f, i == s.flowIdx)
		if x >= curX && x < curX+pillWidth {
			return point.Target{Kind: point.ComposeFlowChoice, Pane: i}
		}

		curX += pillWidth + 1
	}

	newBtn := theme.Pill(" ➕ "+p.T("compose.new_flow_btn", "New")+" ", theme.PillInk, theme.PillCreate)

	newWidth := lipgloss.Width(newBtn)
	if x >= curX && x < curX+newWidth {
		return point.Target{Kind: point.ComposeNewFlow}
	}

	return point.Target{Kind: point.ComposeField, Pane: field}
}

// composeBoxStart is the column the text inside the box is drawn at: the
// box begins where every other value of the form begins, and the border
// with its space takes the two cells after that.
const composeBoxStart = composeLabelStart + 2

// onComposePaste is whether a column is over the paste button, which stands
// one cell past the right edge of the box.
//
// The width comes from the button itself because the words on it are
// translated: a reader whose language spends more cells on "Paste" would
// otherwise have half a button that listens.
func (s State) onComposePaste(x int, e Env) bool {
	from := composeLabelStart + s.composeBoxWidth(e.Frame.Body.W, e) + 1

	return x >= from && x < from+lipgloss.Width(composePasteTab(e.Words))
}

// caretAt is a click inside a field, as the place in the value it points
// at. A click to the left of where the value starts is the start of it,
// which is where a reader who lands on the label meant to be.
func caretAt(field, row, col int) point.Target {
	return point.Target{Kind: point.ComposeCaret, Pane: field, Phase: row, Caret: max(col, 0)}
}
