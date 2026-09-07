package flows

// A loop, in the screen that shows what a flow is made of.
//
// A loop has no engine and no prompt of its own — what runs is inside it — so
// the ordinary phase card drew a name, an engine of "/default" and nothing
// else. Somebody reading that sees a step that does nothing, which is the
// opposite of what a loop is.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// loopCard is one loop: how many turns it gets, what would let it stop, and
// what goes round.
//
// The checks come first among the three. They are the whole of why a loop is
// not a machine for spending a quota window on a wall — the thing that ends
// it is a command's exit code and never a model saying it is finished — so
// they are what a reader most needs to see.
func (s State) loopCard(i int, ph flow.Phase, w int, e Env) []string {
	p := e.Words

	head := fmt.Sprintf("    [%s %d: %s] (%s)",
		p.T("flows.phase_label", "Phase"), i+1, ph.Name,
		p.T("flows.loop_badge", "loop · up to {n} turns", about("n", strconv.Itoa(ph.Loop.Max))))

	out := []string{theme.Paint(theme.Accent).Bold(true).Render(cells.Fit(head, w))}

	names := make([]string, 0, len(ph.Loop.Until))
	for _, g := range ph.Loop.Until {
		names = append(names, g.Name)
	}

	out = append(out, cells.Fit("      "+theme.Paint(theme.Dim).Render(
		p.T("flows.loop_until", "stops when these pass: {checks}",
			about("checks", strings.Join(names, ", ")))), w))

	for n, inner := range ph.Loop.Phases {
		line := fmt.Sprintf("      %d. %s  %s", n+1, inner.Name,
			theme.Paint(theme.Dim).Render(inner.Engine+"/"+cells.OrDef(inner.Model, "default")))
		if inner.FeedOutput {
			line += "  " + theme.Paint(theme.Live).Render(p.T("flows.loop_fed", "reads what failed"))
		}

		out = append(out, cells.Fit(line, w))
	}

	return append(out, "")
}

// loopLine is a loop in the list of flows, where every phase gets one row.
//
// It reads as an empty step without this: a loop names no engine and carries
// no prompt, so the ordinary row drew a number, a name and two blanks. What
// it needs to say in one line is that it goes round, how many times, and what
// would let it stop.
func (s State) loopLine(idx int, ph flow.Phase, e Env) string {
	p := e.Words

	names := make([]string, 0, len(ph.Loop.Until))
	for _, g := range ph.Loop.Until {
		names = append(names, g.Name)
	}

	return fmt.Sprintf("%s%d. %s  %s",
		strings.Repeat(" ", cells.Gutter+2), idx+1,
		theme.Paint(theme.Accent).Render(ph.Name),
		theme.Paint(theme.Dim).Render(p.T("flows.loop_line", "loop ×{n} until: {checks}",
			about("n", strconv.Itoa(ph.Loop.Max)),
			about("checks", strings.Join(names, ", ")))))
}
