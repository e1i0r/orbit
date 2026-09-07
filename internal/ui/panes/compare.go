package panes

// What the both-sides section draws.
//
// It goes first in the pane, because it is the only part of it with an exit
// code behind it: the history below is a pattern and the delta under that is
// a claim, and a reader deciding what to do next should meet the verified
// thing first.

import (
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// compareRows is the section: what it is, and what the two sides answered.
func (e Env) compareRows() []string {
	p := e.Words

	rows := []string{"", e.impactHead(p.T("compare.title", "WHAT THE CHECKS SAY, BOTH SIDES"))}
	rows = append(rows, e.explains(p.T("compare.about",
		"the flow's own checks, run on the branch this work was cut from and on the work itself. An exit code decided every line below — nothing here is anybody's reading."))...)
	rows = append(rows, "")

	switch {
	case e.Reach.Running:
		return append(rows, "  "+e.Spinner+theme.Paint(theme.Live).Render(p.T("compare.out2",
			"running {n} checks on both sides at once… {secs}",
			about("n", strconv.Itoa(len(e.Reach.Offered))),
			about("secs", e.Reach.For.String()))), "")
	case e.Reach.ChecksFailed != "":
		return append(rows, "  "+theme.Paint(theme.Bad).Render(e.Reach.ChecksFailed), "")
	case !e.Reach.Compared:
		return append(rows, e.compareOffer()...)
	case len(e.Reach.Checks) == 0:
		return append(rows, "  "+theme.Paint(theme.Dim).Render(p.T("compare.none_ran", "no checks ran")), "")
	}

	for _, d := range e.Reach.Checks {
		rows = append(rows, e.compareLine(d)...)
	}

	// The key stays on screen after the run, because the answer goes stale
	// the moment the phase writes another line — and a reader who has just
	// read a failure is exactly the one who wants to ask again.
	return append(rows, "  "+theme.Paint(theme.Live).Render(p.T("compare.again", "[r] runs them again")), "")
}

// compareOffer is what it would run, and the key that runs it. It says the
// cost out loud: this is a test suite twice on somebody's machine.
func (e Env) compareOffer() []string {
	p := e.Words

	if e.Gone {
		return nil
	}

	checks := e.Reach.Offered
	if len(checks) == 0 {
		return []string{"  " + theme.Paint(theme.Dim).Render(p.T("compare.no_checks_here",
			"this task's flow carries no checks, so there is nothing to run on either side")), ""}
	}

	rows := []string{"  " + theme.Paint(theme.Live).Render(p.T("compare.offer",
		"[r] runs these on both sides — the base is checked out on its own and thrown away afterwards"))}

	for _, c := range checks {
		rows = append(rows, "    "+theme.Paint(theme.Dim).Render("· "+c.Name+" · "+c.Command))
	}

	return append(rows, "")
}

// compareLine is one check and what each side answered.
func (e Env) compareLine(d repo.Divergence) []string {
	p := e.Words

	if d.Same() {
		return []string{"  " + theme.Paint(theme.OK).Render("✓ ") + theme.Text(theme.Primary).Render(d.Name) + " " +
			theme.Paint(theme.Dim).Render(p.T("compare.same", "the same on both sides"))}
	}

	mark, said := theme.Paint(theme.Bad).Render("✗ "), p.T("compare.broke", "passed before, fails now")
	if d.Fixed() {
		mark, said = theme.Paint(theme.OK).Render("✓ "), p.T("compare.fixed", "failed before, passes now")
	}

	rows := []string{
		"  " + mark + theme.Text(theme.Primary).Render(d.Name) + " " + theme.Paint(theme.Warn).Render(said),
		"    " + theme.Paint(theme.Dim).Render(d.Command),
		"    " + theme.Paint(theme.Dim).Render(p.T("compare.base_said", "base:     {said}", about("said", e.verdict(d.Base)))),
		"    " + theme.Paint(theme.Dim).Render(p.T("compare.now_said", "worktree: {said}", about("said", e.verdict(d.Now)))),
	}

	// The end of what it printed, and not one line of it: a test runner
	// says which test failed a few lines above the word FAIL, and a pane
	// that shows only the last line shows the word and not the reason.
	if !d.Now.Passed() {
		for _, line := range lastLines(d.Now.Out, saidLines) {
			rows = append(rows, "      "+theme.Paint(theme.Dim).Render(line))
		}
	}

	return append(rows, "")
}

// verdict is what one side answered, in words: passed, an exit code, or the
// reason the command could not be run at all.
func (e Env) verdict(r repo.Ran) string {
	p := e.Words

	switch {
	case r.Failed != nil:
		return p.T("compare.could_not_run", "could not be run: {err}", about("err", e.said(r.Failed)))
	case r.Passed():
		return p.T("compare.passed", "passed")
	default:
		return p.T("compare.exited", "exit {code}", about("code", strconv.Itoa(r.Exit)))
	}
}

// saidLines is how much of a failing command's output is shown: enough to
// carry the failing test's name and the assertion under it, and not so much
// that the section becomes the log.
const saidLines = 6

// lastLines is the end of what a command printed, which is where a test
// runner says what failed.
func lastLines(out string, most int) []string {
	var kept []string

	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0 && len(kept) < most; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			kept = append([]string{cells.Fit(line, 120)}, kept...)
		}
	}

	return kept
}
