package settings

// Apply puts one setting into the file, and says what stopped it.
//
// It is a door of its own because two other things write settings: a click
// on a pill, and the engine screen, which sets the engine and its model from
// its own list.

import (
	"slices"

	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// Apply writes one setting down and says what it now is, or why it is not.
//
// The sentence is out of the catalogue rather than concatenated here. It was
// two English strings glued to a value — "autopilot is now on" — and it was
// the only line on a screen whose title, subtitle and every way out of it
// are already in the reader's language.
//
// What comes back from a failed write is not: those sentences name a file or
// a lock and they are English wherever they are raised.
func Apply(name, val string, e Env) Out {
	dials, err := write(name, val, e)
	if err != nil {
		return Out{Said: err.Error()}
	}

	p := e.Words
	out := Out{Dials: dials}

	if name == "language" {
		out.Said = p.T("settings.language_changed", "language changed to {lang}", words.Arg{Name: "lang", Value: val})
		out.Lang = val

		return out
	}

	out.Said = p.T("settings.now", "{key} is now {val}",
		words.Arg{Name: "key", Value: name}, words.Arg{Name: "val", Value: val})

	return out
}

// Blank puts one setting back to what Orbit ships.
//
// Through write, like every other change on this screen, because putting a
// setting back is choosing the value it came with: the theme has to repaint,
// the catalogue has to reload, and a dial the file does not hold has to be
// let go of. A clear that wrote straight to the file would leave the window
// drawing what it used to be.
//
// The sentence is its own. "autopilot is now off" and "autopilot is back to
// off" are the same fact and different answers — one says somebody chose it,
// and the other that nobody has.
func Blank(name string, e Env) Out {
	if e.Store == nil {
		return Out{}
	}

	val := e.Store.Fresh(name)

	dials, err := write(name, val, e)
	if err != nil {
		return Out{Said: err.Error()}
	}

	out := Out{Dials: dials, Said: e.Words.T("settings.back_to", "{key} is back to {val}",
		words.Arg{Name: "key", Value: name}, words.Arg{Name: "val", Value: shown(e, val)})}

	if name == "language" {
		out.Lang = val
	}

	return out
}

// shown is a value as a row shows it, which for a setting that comes as
// nothing is a word and not an empty space. A line ending in "is back to"
// reads as a sentence that broke off.
func shown(e Env, val string) string {
	if val == "" {
		return e.Words.T("settings.nothing", "nothing")
	}

	return val
}

// write puts one setting down, and answers what stopped it.
//
// The new value is already drawn by the time this runs, so a write that
// fails silently is a window showing a setting that is not in the file, and
// the next time the reader opens it the switch has flipped itself back. The
// settings file has a lock, so these can also refuse, in words, after
// waiting two seconds for a second orbit to finish changing something.
//
// What comes back is the dials, when one of them moved: effort and thinking
// are not in the file at all, and the window is what keeps them.
func write(name, val string, e Env) (*Dials, error) {
	// The two the settings file does not hold. They never reach Choose,
	// because there is nothing for it to write them into.
	switch name {
	case "effort":
		d := e.Dials
		d.Effort = val

		return &d, nil
	case "thinking":
		d := e.Dials
		d.Thinking = val

		return &d, nil
	}

	if e.Choose == nil {
		return nil, nil
	}

	// Everything else goes through the validator declared beside the
	// setting. The screen used to carry its own copy of one of those — the
	// two checks on the unread cap, in the same words, out of the same
	// catalogue — and none of the other twelve, so a number the command
	// line refused, the window wrote.
	if err := e.Choose(name, val); err != nil {
		return nil, err
	}

	// What is left is what a value being written *also* does to the window,
	// which is not the file's business and cannot be declared beside it.
	switch name {
	case "theme":
		theme.SetCurrentTheme(val)
	case "engine":
		return followed(val, e)
	}

	return nil, nil
}

// followed moves the model and the effort when the engine they were chosen
// for is no longer the one selected.
//
// It runs after the engine is written rather than instead of writing it: a
// model belongs to an engine, and leaving one engine's model selected under
// another is a phase asking for something nobody has.
func followed(val string, e Env) (*Dials, error) {
	models, _ := e.Models(val)
	if len(models) > 0 && !slices.Contains(models, modelNow(e)) {
		if err := e.Choose("model", models[0]); err != nil {
			return nil, err
		}
	}

	d := e.Dials

	efforts, _ := e.Efforts(val)
	if !slices.Contains(efforts, d.Effort) && len(efforts) > 0 {
		d.Effort = efforts[0]

		return &d, nil
	}

	return nil, nil
}

// modelNow is the model the file holds, read back off the table rather than
// through a getter of its own.
func modelNow(e Env) string {
	if e.Kept == nil {
		return ""
	}

	for _, one := range e.Kept() {
		if one.Name == "model" {
			return one.Value
		}
	}

	return ""
}
