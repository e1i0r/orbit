package ui

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
)

// loopCard is one loop: how many turns it gets, what would let it stop, and
// what goes round.
//
// The checks come first among the three. They are the whole of why a loop is
// not a machine for spending a quota window on a wall — the thing that ends
// it is a command's exit code and never a model saying it is finished — so
// they are what a reader most needs to see.
func (m Model) loopCard(i int, ph flow.Phase, w int) []string {
	p := m.opts.Words

	head := fmt.Sprintf("    [%s %d: %s] (%s)",
		p.T("flows.phase_label", "Phase"), i+1, ph.Name,
		p.T("flows.loop_badge", "loop · up to {n} turns", about("n", strconv.Itoa(ph.Loop.Max))))

	out := []string{Paint(Accent).Bold(true).Render(fit(head, w))}

	names := make([]string, 0, len(ph.Loop.Until))
	for _, g := range ph.Loop.Until {
		names = append(names, g.Name)
	}

	out = append(out, fit("      "+Paint(Dim).Render(
		p.T("flows.loop_until", "stops when these pass: {checks}",
			about("checks", strings.Join(names, ", ")))), w))

	for n, inner := range ph.Loop.Phases {
		line := fmt.Sprintf("      %d. %s  %s", n+1, inner.Name,
			Paint(Dim).Render(inner.Engine+"/"+orDef(inner.Model, "default")))
		if inner.FeedOutput {
			line += "  " + Paint(Live).Render(p.T("flows.loop_fed", "reads what failed"))
		}

		out = append(out, fit(line, w))
	}

	return append(out, "")
}

// loopLine is a loop in the list of flows, where every phase gets one row.
//
// It reads as an empty step without this: a loop names no engine and carries
// no prompt, so the ordinary row drew a number, a name and two blanks. What
// it needs to say in one line is that it goes round, how many times, and what
// would let it stop.
func (m Model) loopLine(idx int, ph flow.Phase) string {
	p := m.opts.Words

	names := make([]string, 0, len(ph.Loop.Until))
	for _, g := range ph.Loop.Until {
		names = append(names, g.Name)
	}

	return fmt.Sprintf("%s%d. %s  %s",
		strings.Repeat(" ", gutter+2), idx+1,
		Paint(Accent).Render(ph.Name),
		Paint(Dim).Render(p.T("flows.loop_line", "loop ×{n} until: {checks}",
			about("n", strconv.Itoa(ph.Loop.Max)),
			about("checks", strings.Join(names, ", ")))))
}
