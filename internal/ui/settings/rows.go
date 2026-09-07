package settings

// Rows is every row of the table, with what it is set to and what it offers.
//
// It is a door because the window points at it: a click lands on a row and
// on one of its pills, and the mouse needs the same list the drawing used.

import (
	"strconv"

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
	if e.Store == nil {
		return nil
	}

	p, st := e.Words, reader(e.Store)

	autopilot := "off"
	if st.Autopilot() {
		autopilot = "on"
	}

	engines := e.Engines()
	engine := cells.OrDef(st.Engine(), cells.First(engines))
	models, modelLabels := e.Models(engine)
	efforts, effortLabels := e.Efforts(engine)

	// The flows are the build's and the reader's, not this screen's. Names
	// written out by hand on this dial leave every flow they do not list —
	// the ones shipped inside the binary and the ones the reader wrote for
	// themselves — impossible to choose here at all.
	flows := s.flows
	if len(flows) == 0 {
		flows = flow.BuiltinNames()
	}

	return []Row{
		{
			Key: "language", Val: cells.OrDef(st.Language(), "en"),
			Options: []string{"en", "es"},
			About:   p.T("setting.language", "the language orbit speaks"),
		},
		{
			Key: "autopilot", Val: autopilot,
			Options: []string{"off", "on"},
			About: p.T("setting.autopilot",
				"whether a run walks its whole flow without stopping"),
		},
		{
			Key: "unread-cap", Val: strconv.Itoa(st.UnreadCap()),
			Options: []string{"0", "3", "5", "10", "20"},
			About: p.T("setting.unread_cap",
				"how many finished tasks may sit unread before nothing new starts"),
		},
		{
			Key: "engine", Val: engine, Options: engines,
			About: p.T("setting.engine", "the engine a task runs on when it names none"),
		},
		{
			Key: "model", Val: chosen(st.Model(), models),
			Options: models, Labels: modelLabels,
			About: p.T("setting.model", "the model a phase asks for when it names none"),
		},
		{
			Key: "effort", Val: chosen(e.Dials.Effort, efforts),
			Options: efforts, Labels: effortLabels,
			About: p.T("setting.effort",
				"the default reasoning effort level for engine sessions"),
		},
		{
			Key: "thinking", Val: cells.OrDef(e.Dials.Thinking, "adaptive"),
			Options: []string{"adaptive", "on", "off"},
			About: p.T("setting.thinking",
				"whether extended thinking mode is enabled for the engine"),
		},
		{
			Key: "flow", Val: cells.OrDef(st.Flow(), flow.Default), Options: flows,
			About: p.T("setting.flow", "the flow a new task is written against"),
		},
		{
			Key: "theme", Val: cells.OrDef(st.Theme(), theme.DefaultTheme),
			Options: theme.AvailableThemes(),
			About:   p.T("setting.theme", "the visual color theme for the window"),
		},
	}
}

// chosen is what a dial is set to, falling to the first option when what it
// holds is not one this build offers.
func chosen(val string, options []string) string {
	for _, o := range options {
		if o == val {
			return val
		}
	}

	if len(options) > 0 {
		return options[0]
	}

	return val
}
