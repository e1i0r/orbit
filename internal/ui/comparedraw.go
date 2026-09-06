package ui

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
)

// compareRows is the section: what it is, and what the two sides answered.
func (m Model) compareRows() []string {
	p := m.opts.Words

	rows := []string{"", m.impactHead(p.T("compare.title", "WHAT THE CHECKS SAY, BOTH SIDES"))}
	rows = append(rows, m.explains(p.T("compare.about",
		"the flow's own checks, run on the branch this work was cut from and on the work itself. An exit code decided every line below — nothing here is anybody's reading."))...)
	rows = append(rows, "")

	switch {
	case m.weigh.running:
		return append(rows, "  "+m.spinner(Live)+Paint(Live).Render(p.T("compare.out2",
			"running {n} checks on both sides at once… {secs}",
			about("n", strconv.Itoa(m.checksNow())),
			about("secs", m.comparedFor().String()))), "")
	case m.weigh.checksErr != nil:
		return append(rows, "  "+Paint(Bad).Render(m.errSaid(m.weigh.checksErr)), "")
	case !m.weigh.checksKnown:
		return append(rows, m.compareOffer()...)
	case len(m.weigh.checks) == 0:
		return append(rows, "  "+Paint(Dim).Render(p.T("compare.none_ran", "no checks ran")), "")
	}

	for _, d := range m.weigh.checks {
		rows = append(rows, m.compareLine(d)...)
	}

	// The key stays on screen after the run, because the answer goes stale
	// the moment the phase writes another line — and a reader who has just
	// read a failure is exactly the one who wants to ask again.
	return append(rows, "  "+Paint(Live).Render(p.T("compare.again", "[r] runs them again")), "")
}

// compareOffer is what it would run, and the key that runs it. It says the
// cost out loud: this is a test suite twice on somebody's machine.
func (m Model) compareOffer() []string {
	p := m.opts.Words

	t, held := m.task(m.detail)
	if !held {
		return nil
	}

	checks := m.checksOf(t)
	if len(checks) == 0 {
		return []string{"  " + Paint(Dim).Render(p.T("compare.no_checks_here",
			"this task's flow carries no checks, so there is nothing to run on either side")), ""}
	}

	rows := []string{"  " + Paint(Live).Render(p.T("compare.offer",
		"[r] runs these on both sides — the base is checked out on its own and thrown away afterwards"))}

	for _, c := range checks {
		rows = append(rows, "    "+Paint(Dim).Render("· "+c.Name+" · "+c.Command))
	}

	return append(rows, "")
}

// compareLine is one check and what each side answered.
func (m Model) compareLine(d repo.Divergence) []string {
	p := m.opts.Words

	if d.Same() {
		return []string{"  " + Paint(OK).Render("✓ ") + Text(Primary).Render(d.Name) + " " +
			Paint(Dim).Render(p.T("compare.same", "the same on both sides"))}
	}

	mark, said := Paint(Bad).Render("✗ "), p.T("compare.broke", "passed before, fails now")
	if d.Fixed() {
		mark, said = Paint(OK).Render("✓ "), p.T("compare.fixed", "failed before, passes now")
	}

	rows := []string{
		"  " + mark + Text(Primary).Render(d.Name) + " " + Paint(Warn).Render(said),
		"    " + Paint(Dim).Render(d.Command),
		"    " + Paint(Dim).Render(p.T("compare.base_said", "base:     {said}", about("said", verdict(m, d.Base)))),
		"    " + Paint(Dim).Render(p.T("compare.now_said", "worktree: {said}", about("said", verdict(m, d.Now)))),
	}

	// The end of what it printed, and not one line of it: a test runner
	// says which test failed a few lines above the word FAIL, and a pane
	// that shows only the last line shows the word and not the reason.
	if !d.Now.Passed() {
		for _, line := range lastLines(d.Now.Out, saidLines) {
			rows = append(rows, "      "+Paint(Dim).Render(line))
		}
	}

	return append(rows, "")
}

// verdict is what one side answered, in words: passed, an exit code, or the
// reason the command could not be run at all.
func verdict(m Model, r repo.Ran) string {
	p := m.opts.Words

	switch {
	case r.Failed != nil:
		return p.T("compare.could_not_run", "could not be run: {err}", about("err", m.errSaid(r.Failed)))
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
			kept = append([]string{fit(line, 120)}, kept...)
		}
	}

	return kept
}

// checksNow is how many checks the run that is out is running.
func (m Model) checksNow() int {
	t, held := m.task(m.detail)
	if !held {
		return 0
	}

	return len(m.checksOf(t))
}
