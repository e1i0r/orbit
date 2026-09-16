package ui

// What a row says about an engine that has nothing left to spend.
//
// Three rows and one subject: the engine ran out, nobody has anything left,
// and — in both cases — when an allowance comes back. They are together
// because the hour is the hard half of all three and is read differently in
// each: from the window's own quota reading for a task whose engine is
// known, and from the record for a task that stopped because nothing was
// free.
//
// Apart from cells.go because that file is the shape of a row, and this is
// what one particular kind of row says.

import (
	"time"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// ranOutWord is a run whose engine had nothing left, and when it comes back.
//
// The second half is the one somebody actually wants: told an engine ran
// out, the next question is always how long, and the window already has the
// answer — it draws it in the header on every frame. Saying it here saves
// the reader going to look for the number they were about to be sent to.
//
// An engine nobody can read a window for says only that it ran out, which is
// still the thing that matters: an engine with no reading is not an engine
// with nothing left.
func (m Model) ranOutWord(t view.Task) string {
	p := m.opts.Words

	said := p.T("reason.ran_out", "{engine} ran out: {phase}", reasonArgs(t.Reason)...)
	if m.opts.Quota == nil || t.Engine == "" {
		return said
	}

	back := time.Duration(0)

	for _, w := range m.opts.Quota(t.Engine).Windows {
		if w.ResetsIn > 0 && (back == 0 || w.ResetsIn < back) {
			back = w.ResetsIn
		}
	}

	if back == 0 {
		return said
	}

	return said + cells.Dot + p.T("reason.ran_out_back", "back in {when}",
		words.Arg{Name: "when", Value: cells.Elapsed(m.now, m.now.Add(-back))})
}

// noEngineWord is a run with nowhere to go, and when somewhere opens.
//
// The hour is the whole point of the row, so it is said when the record has
// it and left out when it does not — a reader told "no engines" with no hour
// is being told the task is dead, and it is not: allowances come back.
func (m Model) noEngineWord(t view.Task) string {
	p := m.opts.Words

	said := backIn(m.now, t.Reason)
	if said == "" {
		return p.T("reason.no_engine", "{engine} ran out: {phase} · no engine has anything left",
			reasonArgs(t.Reason)...)
	}

	args := reasonArgs(t.Reason)
	for i := range args {
		if args[i].Name == "back" {
			args[i].Value = said
		}
	}

	return p.T("reason.no_engine_back", "{engine} ran out: {phase} · no engine for another {back}", args...)
}

// backIn is how long the reason says until an allowance returns, said the
// way the row above says it, and nothing at all when the record did not say.
//
// The record keeps the exact duration because whatever reads it next should
// not have to parse prose; a row is read by a person, and "1h35m0s" is a
// machine talking. cells.Elapsed is the same spelling the engine's own
// "back in" uses, so the two rows about waiting for an allowance wait in the
// same words.
func backIn(now time.Time, r view.Reason) string {
	for _, a := range r.Args {
		if a.Name != "back" {
			continue
		}

		d, err := time.ParseDuration(a.Value)
		if err != nil || d <= 0 {
			return ""
		}

		return cells.Elapsed(now, now.Add(-d))
	}

	return ""
}
