package settings

// Rows is every row of the table, with what it is set to and what it offers.
//
// It is a door because the window points at it: a click lands on a row and
// on one of its pills, and the mouse needs the same list the drawing used.
//
// The list is read off the vocabulary rather than written out here. It used
// to be written out here, and every setting added after it was written went
// into internal/verb and not into this file — six of thirteen, by the time
// anybody counted, including whether Orbit may interrupt you and which
// account may command it over a chat. What this file still decides is what a
// row *offers*, which is the one thing the vocabulary cannot know: the
// engines this build has, the models that engine has, the flows on this
// machine.

import (
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// Row is one setting: its name, what it is now, what it offers, and the
// sentence under it.
type Row struct {
	Key     string
	Val     string
	Options []string
	// Labels is what each option is drawn as, when that is not the option
	// itself. Only the model dial needs it — opencode's models are stored
	// provider-qualified and shown without the provider — and every other
	// row leaves it nil.
	Labels []string
	About  string
}

// Label is what the option at i is drawn as.
func (r Row) Label(i int) string { return cells.DialLabel(r.Options, r.Labels, i) }

// Rows is the table as it stands right now.
func (s State) Rows(e Env) []Row {
	if e.Kept == nil {
		return nil
	}

	out := make([]Row, 0, len(theDials))

	for _, one := range e.Kept() {
		out = append(out, s.dial(one, e))

		// The two the settings file does not hold sit beside the dial they
		// belong to. Effort is the engine's, so it follows the model; the
		// alternative is a table where the four choices a run is made with
		// are not next to each other.
		if one.Name == "model" {
			out = append(out, s.window(e)...)
		}
	}

	return out
}

// dial is one declared setting as a row: the sentence and the value are the
// vocabulary's, and what it offers is this build's.
func (s State) dial(one Kept, e Env) Row {
	ids, labels := s.offers(one.Name, e)

	return Row{
		Key:     one.Name,
		Val:     chosen(one.Value, ids),
		Options: ids,
		Labels:  labels,
		About:   one.About,
	}
}

// window is effort and thinking, which are not in the settings file at all
// and are the window's own to keep. They are rows like the rest because a
// reader turning them does not care where the value is written down.
func (s State) window(e Env) []Row {
	p := e.Words
	efforts, effortLabels := e.Efforts(s.engine(e))

	return []Row{{
		Key: "effort", Val: chosen(e.Dials.Effort, efforts),
		Options: efforts, Labels: effortLabels,
		About: p.T("setting.effort", "the default reasoning effort level for engine sessions"),
	}, {
		Key: "thinking", Val: cells.OrDef(e.Dials.Thinking, "adaptive"),
		Options: []string{"adaptive", "on", "off"},
		About: p.T("setting.thinking",
			"whether extended thinking mode is enabled for the engine"),
	}}
}

// theDials is what each setting offers that has anything to offer, and the
// length Rows reserves against.
//
// A setting absent from it is a row typed into rather than chosen from,
// which is the right answer for a chat id, a budget and a percentage — no
// list of numbers is the list somebody wants. Nothing here is a second copy
// of what a setting accepts: the vocabulary still refuses what it will not
// have, and these are the answers worth putting under a cursor.
var theDials = map[string][]string{
	"language":     {"en", "es"},
	"autopilot":    {"off", "on"},
	"unread-cap":   {"0", "3", "5", "10", "20"},
	"check-record": {"off", "on"},
	"notify":       {"off", "on"},
	"quota-floor":  {"0", "10", "25", "50"},
	"run-timeout":  {"0", "30m", "1h", "2h", "4h"},
	// The decision engine, and how sure it has to be. Shadow sits between
	// the two because it is how a reader finds out whether to trust it.
	"decisions":      {"off", "shadow", "on"},
	"decision-floor": {"60", "70", "80", "90"},
	"effort":         nil,
	"thinking":       nil,
}

// offers is what one row puts under the cursor, and the labels for them when
// the option and the word are not the same.
func (s State) offers(name string, e Env) (ids, labels []string) {
	switch name {
	case "engine":
		return e.Engines(), nil
	case "model":
		return e.Models(s.engine(e))
	case "flow":
		// The flows are the build's and the reader's, not this screen's.
		// Names written out by hand on this dial leave every flow they do
		// not list — the ones shipped inside the binary and the ones the
		// reader wrote for themselves — impossible to choose here at all.
		if len(s.flows) > 0 {
			return s.flows, nil
		}

		return flow.BuiltinNames(), nil
	case "theme":
		return theme.AvailableThemes(), nil
	}

	return theDials[name], nil
}

// engine is the engine the model and effort dials are read against: what the
// file holds, or the first this build has when it holds nothing.
func (s State) engine(e Env) string {
	for _, one := range e.Kept() {
		if one.Name == "engine" {
			return cells.OrDef(one.Value, cells.First(e.Engines()))
		}
	}

	return cells.First(e.Engines())
}

// chosen is what a dial is set to, falling to the first option when what it
// holds is not one this build offers.
//
// A setting with nothing to offer keeps whatever it holds: a chat id is not
// wrong for being absent from a list there is no list of.
func chosen(val string, options []string) string {
	if len(options) == 0 {
		return val
	}

	for _, o := range options {
		if o == val {
			return val
		}
	}

	return options[0]
}
