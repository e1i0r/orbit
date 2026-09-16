package known

// The four bands, and which one a rule is in.
//
// A band is what you have to do about what is in it, which is the board's own
// idea and the reason that screen reads at a glance. The question a reader
// opens this screen with is "is there anything here for me", and the answer
// is where a row is rather than a word inside it.

import (
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// band is one heading of the list.
type band int

// The bands, in the order they are drawn: what wants an answer first, then
// what is working, then what is not being told to anybody.
const (
	waitsOnYou band = iota
	refuses
	onlySays
	silent
)

// bands is that order, for a caller walking them.
var bands = []band{waitsOnYou, refuses, onlySays, silent}

// bandName is the heading, with the glyph the board would have given it.
func bandName(b band, e Env) string {
	p := e.Words

	switch b {
	case waitsOnYou:
		return "🛑 " + p.T("knowledge.band_waiting", "WAITING ON YOU")
	case refuses:
		return "⚡ " + p.T("knowledge.band_stops", "STOP THE WORK")
	case onlySays:
		return "💬 " + p.T("knowledge.band_says", "JUST SAID, BEFORE EVERY RUN")
	}

	return "😴 " + p.T("knowledge.band_silent", "TOLD TO NOBODY")
}

// bandCount paints the number over a band, and only one of the four is ever
// worth a colour: how many things are waiting on a person.
func bandCount(b band) theme.Role {
	if b == waitsOnYou {
		return theme.Warn
	}

	return theme.Dim
}

// bandOf is where a rule belongs.
//
// Waiting first, whatever else is true of it: a rule somebody sent back to
// be decided about is a question, and a question filed under what it happens
// to do meanwhile is a question nobody answers.
func bandOf(f knowledge.Fact) band {
	switch {
	case f.Review:
		return waitsOnYou
	case !f.Tells():
		return silent
	case f.Action() == knowledge.Stops:
		return refuses
	}

	return onlySays
}

// entry is one row the cursor can be on: a sentence in the tray, or a rule.
type entry struct {
	said bool
	at   int
}

// inBand is what is under one heading, in the order it was read.
func (s State) inBand(b band) []entry {
	var out []entry

	// Every sentence in the tray is a question, so they are all in the
	// first band, above the rules that are only waiting to be looked at
	// again. Nothing has been decided about them at all.
	if b == waitsOnYou {
		for i := range s.waiting {
			out = append(out, entry{said: true, at: i})
		}
	}

	for i, f := range s.facts {
		if bandOf(f) == b {
			out = append(out, entry{at: i})
		}
	}

	return out
}

// order is every row the cursor can be on, in the order the bands draw them.
// It is what s.sel indexes: the cursor walks the screen, not the store.
func (s State) order() []entry {
	var out []entry

	for _, b := range bands {
		out = append(out, s.inBand(b)...)
	}

	return out
}

// onSaid is the sentence under the cursor, and false when the cursor is on a
// rule instead.
func (s State) onSaid() (Said, bool) {
	all := s.order()
	if s.sel < 0 || s.sel >= len(all) || !all[s.sel].said {
		return Said{}, false
	}

	return s.waiting[all[s.sel].at], true
}

// onFact is the rule under the cursor, and false when the cursor is on a
// sentence in the tray instead.
func (s State) onFact() (knowledge.Fact, bool) {
	all := s.order()
	if s.sel < 0 || s.sel >= len(all) || all[s.sel].said {
		return knowledge.Fact{}, false
	}

	return s.facts[all[s.sel].at], true
}
