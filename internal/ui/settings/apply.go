package settings

// Apply puts one setting into the file, and says what stopped it.
//
// It is a door of its own because two other things write settings: a click
// on a pill, and the engine screen, which sets the engine and its model from
// its own list.

import (
	"errors"
	"slices"
	"strconv"

	"github.com/e1i0r/orbit/internal/flow"
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
	if e.Store == nil {
		return nil, nil
	}

	var st writer = e.Store

	switch name {
	case "language":
		return nil, st.SetLanguage(val)
	case "autopilot":
		return nil, st.SetAutopilot(val == "on")
	case "unread-cap":
		return nil, unreadCap(val, e)
	case "engine":
		return engine(val, e)
	case "model":
		return nil, st.SetModel(val)
	case "effort":
		d := e.Dials
		d.Effort = val

		return &d, nil
	case "thinking":
		d := e.Dials
		d.Thinking = val

		return &d, nil
	case "flow":
		// A name that is a path could never be a flow in any future, and
		// it is the one thing `orbit set` refused that this screen did
		// not — it typed the same value into the same file without the
		// check, so what the command line would not take, the window did.
		if err := flow.ValidName(val); err != nil {
			return nil, err
		}

		return nil, st.SetFlow(val)
	case "theme":
		if err := st.SetTheme(val); err != nil {
			return nil, err
		}

		theme.SetCurrentTheme(val)
	}

	return nil, nil
}

// unreadCap is the one setting a number is typed into, and the two ways that
// goes wrong.
//
// A value that was not a number was dropped on the floor and the band still
// said "unread-cap is now lots". A negative one was written, and both this
// window and internal/task read anything but a positive number as no cap at
// all — so typing -1 turned the brake off while looking like it set one.
func unreadCap(val string, e Env) error {
	p := e.Words

	n, err := strconv.Atoi(val)
	if err != nil {
		return errors.New(p.T("settings.not_a_number", "{val} is not a whole number", words.Arg{Name: "val", Value: val}))
	}

	if n < 0 {
		return errors.New(p.T("settings.negative_cap", "the unread cap cannot be negative; zero is no cap at all"))
	}

	return e.Store.SetUnreadCap(n)
}

// engine also moves the model and the effort when the engine they were
// chosen for is no longer the one selected.
func engine(val string, e Env) (*Dials, error) {
	st := e.Store
	if err := st.SetEngine(val); err != nil {
		return nil, err
	}

	models, _ := e.Models(val)
	if !slices.Contains(models, st.Model()) && len(models) > 0 {
		if err := st.SetModel(models[0]); err != nil {
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
