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

// The five states a rule can be in, in the order the list draws them: what
// wants an answer first, then what is working, then what is not.
//
// Five words and one set of them. The band a rule is listed under and the
// word on its own screen are the same word, because they are the same
// question — a screen that said "told to nobody" over a rule whose card said
// "switched off" was two vocabularies for one fact, and the reader had to
// hold both.
const (
	waits band = iota
	stops
	says
	paused
	off
)

// bands is that order, for a caller walking them.
var bands = []band{waits, stops, says, paused, off}

// bandName is the state, with the glyph the board would have given it.
func bandName(b band, e Env) string { return bandMark(b) + " " + stateName(b, e) }

// bandMark is the glyph beside a state, which is how the eye finds the band
// it is looking for before it has read anything.
func bandMark(b band) string {
	switch b {
	case waits:
		return "🛑"
	case stops:
		return "⚡"
	case says:
		return "💬"
	case paused:
		return "😴"
	}

	return "🚫"
}

// stateName is the one word for a state, wherever it is said.
func stateName(b band, e Env) string {
	p := e.Words

	switch b {
	case waits:
		return p.T("knowledge.band_waiting", "WAITING")
	case stops:
		return p.T("knowledge.band_stops", "BLOCKS")
	case says:
		return p.T("knowledge.band_says", "SAYS")
	case paused:
		return p.T("knowledge.band_paused", "PAUSED")
	}

	return p.T("knowledge.band_off", "OFF")
}

// bandRole is the colour a state is said in, and it is the same colour
// wherever it is said.
func bandRole(b band) theme.Role {
	switch b {
	case waits:
		return theme.Warn
	case stops:
		return theme.Bad
	case says:
		return theme.Live
	case paused:
		return theme.Warn
	}

	return theme.Dim
}

// bandCount paints the number over a band, and only one of the five is ever
// worth a colour: how many things are waiting on a person.
func bandCount(b band) theme.Role {
	if b == waits {
		return theme.Warn
	}

	return theme.Dim
}

// bandOf is where a rule belongs, which is the one question its record
// folds down to. The fold is internal/knowledge's, so this screen and the
// browser cannot disagree about a rule they are both looking at.
func bandOf(f knowledge.Fact) band {
	switch f.Standing() {
	case knowledge.Waiting:
		return waits
	case knowledge.Blocks:
		return stops
	case knowledge.Says:
		return says
	case knowledge.Stopped:
		return paused
	}

	return off
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
	if b == waits {
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
