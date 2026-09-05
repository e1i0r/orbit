package ui

// A dial with more choices than fit on a row.
//
// opencode answers to sixty-odd models. Drawn as one pill per model, that
// row is four times the width of the window, and fit cuts it — so every
// model past the cut was invisible, and a reader looking for gpt-5.4-pro saw
// an ellipsis where it should have been and concluded Orbit could not run
// it. What is drawn now is a window around the one that is chosen, with the
// count beside it, so walking the dial walks the whole catalogue.

import "strconv"

// dialShown is how many choices a windowed dial draws at once.
const dialShown = 5

// dialValue is one dial's row: the chosen option, its neighbours, and how
// many are on either side.
func (m Model) dialValue(field int, ids, labels []string, current string) string {
	if len(ids) <= dialShown {
		return renderComboPillsLabelled(ids, labels, current)
	}

	at := 0

	for i, id := range ids {
		if id == current {
			at = i
			break
		}
	}

	from := max(min(at-dialShown/2, len(ids)-dialShown), 0)
	to := from + dialShown

	row := renderComboPillsLabelled(ids[from:to], labels[min(from, len(labels)):min(to, len(labels))], current)

	if from > 0 {
		row = Paint(Dim).Render("‹"+strconv.Itoa(from)+" ") + row
	}

	if to < len(ids) {
		row += Paint(Dim).Render(" " + strconv.Itoa(len(ids)-to) + "›")
	}

	return row + " " + Paint(Dim).Render("("+strconv.Itoa(at+1)+"/"+strconv.Itoa(len(ids))+")")
}
