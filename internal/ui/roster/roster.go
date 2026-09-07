// Package roster is the engines this build can run, their dials, and what is
// left of each one's quota.
//
// It is the vocabulary the window's ports answer in, and it is a package of
// its own because the screens that draw it — the engine knobs, the quota
// screen, the header's chip — are packages of their own: a type a screen
// cannot name is a type it cannot be handed.
package roster

import (
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/words"
)

// Reading is what the window learns about one engine's quota.
//
// Money and Sourced are carried as answers rather than as the billing mode
// they were derived from, because the mode is not this package's to read:
// internal/quota decides what a number about an engine means, and the window
// is told the outcome. Sourced is separate from a window count for the
// difference it protects — an engine nobody can read a window for is not an
// engine with no window left.
type Reading struct {
	Engine  string
	Money   bool
	Sourced bool
	Windows []Window
}

// Window is what the window learns about remaining quota.
type Window struct {
	Key      string
	Label    string
	Pct      float64
	ResetsIn time.Duration
}

// Engine is what the window knows about an engine's dials and setup.
//
// Setup is a function of a printer for the reason Command.About is: the
// steps are sentences a reader reads, so they go through internal/words like
// every other line on this screen, and they follow a language changed after
// this slice was handed over.
type Engine struct {
	Name      string
	Available bool
	Setup     func(*words.Printer) []string
	Models    []Choice
	Efforts   []Choice
	CanThink  bool
}

// Choice is one selectable value for an engine dial.
type Choice struct {
	ID    string
	Label string
}

// Used is the share of a window already spent, capped at the whole of it:
// a proxy reporting more used than there was is reporting an overage, and
// 140% of a window drawn as a bar is a bar with nowhere to go.
func Used(w Window) float64 {
	if w.Pct > 100 {
		return 100
	}

	return w.Pct
}

// Says is one quota window as a reader reads it.
func Says(p *words.Printer, w Window) string {
	return p.T("status.quota", "{pct} used in {label} · resets in {resets}",
		arg("pct", fmt.Sprintf("%.0f%%", Used(w))),
		arg("label", w.Label),
		arg("resets", resetIn(w.ResetsIn)),
	)
}

func resetIn(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}

	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}

	return fmt.Sprintf("%ds", s)
}

// Spent is a reading's windows on one line: the share used of each,
// and the word said once at the end.
//
// Empty for a reading with nothing in it, which is what a caller draws when
// there is nothing to draw rather than a sentence about absence — the one
// place that says a source is missing is the status line, once.
func Spent(p *words.Printer, reading Reading) string {
	var parts []string

	for _, w := range reading.Windows {
		parts = append(parts, fmt.Sprintf("%.0f%% %s", Used(w), w.Label))
	}

	if len(parts) == 0 {
		return ""
	}

	// The word is said once, at the end, and it is the word the providers
	// own screens use. A percentage on a status bar is read as whichever of
	// the two numbers the reader expects, and the two are opposites: 1% of a
	// window is either almost nothing spent or almost nothing left. Saying
	// used keeps this chip and the /usage screen a reader has open in
	// another terminal reporting the same figure rather than its complement.
	return p.T("header.quota_used", "{windows} used",
		arg("windows", strings.Join(parts, " · ")))
}

// Quiet is what a reading with no window of its own says instead.
//
// Three ways to have no percentage, and a screen that drew the same blank
// for all of them would be the silence a quota was read to end. An engine
// paid per token has no window to be at the end of; one with a source that
// has answered nothing is a source to go and look at — a base URL a proxy
// does not serve /quota on reads exactly like an engine with no proxy, and
// only one of those is worth fixing; and an engine with nowhere to look at
// all says so, as it does on the status line.
func Quiet(p *words.Printer, reading Reading) string {
	switch {
	case reading.Money:
		return p.T("quota.per_token", "billed per token")
	case reading.Sourced:
		return p.T("quota.silent", "source answered nothing")
	default:
		return p.T("status.quota_unread", "no quota source for {engine}",
			arg("engine", reading.Engine))
	}
}

// arg names a value and what the sentence calls it.
func arg(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}
