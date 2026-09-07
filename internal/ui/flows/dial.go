package flows

// A dial with more choices than fit on a row.
//
// opencode answers to sixty-odd models. Drawn as one pill per model, that
// row is four times the width of the window, and fit cuts it — so every
// model past the cut was invisible, and a reader looking for gpt-5.4-pro saw
// an ellipsis where it should have been and concluded Orbit could not run
// it. What is drawn now is a window around the one that is chosen, with the
// count beside it, so walking the dial walks the whole catalogue.

import (
	"strconv"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// dialShown is how many choices still read as a row of pills. Past it, the
// row says what is in use and where the rest are.
const dialShown = 5

// dialValue is one dial's row.
//
// For a short list it is the pills, which are the fastest thing to read and
// to click. For a long one it is the single value in use and a count: agy
// answers to fourteen models with names like "Gemini 3.8 Flash (Medium)",
// and a row of fourteen of those is a wall three screens wide that says
// nothing about which one is set — the reader in front of it could not tell
// what was chosen, and had to press right once per model to find out.
func (s State) dialValue(field int, ids, labels []string, current string, e Env) string {
	if len(ids) <= dialShown {
		return renderComboPillsLabelled(ids, labels, current)
	}

	at, shown := -1, e.Words.T("flows.dial_default", "default")

	for i, id := range ids {
		if id == current {
			at, shown = i, cells.DialLabel(ids, labels, i)
			break
		}
	}

	where := e.Words.T("flows.dial_of", "{n} of {total}",
		about("n", strconv.Itoa(at+1)), about("total", strconv.Itoa(len(ids))))
	if at < 0 {
		where = e.Words.T("flows.dial_count", "{total} to choose from",
			about("total", strconv.Itoa(len(ids))))
	}

	return theme.Paint(theme.Sel).Render(" "+shown+" ") + " " + theme.Paint(theme.Dim).Render(where+" · ") +
		theme.Paint(theme.Live).Render(e.Words.T("flows.dial_open", "[↵] see them all"))
}
