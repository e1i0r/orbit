// Package engines is the engine and model knobs: which engine a run goes to,
// which of its models, how hard it thinks, and the setup steps for one this
// machine cannot run yet.
//
// It is entered through this file. State is the screen and the dials it
// holds while it is up, Env is the catalogue and the room the window lends
// it, and Out is what it asks for when a gesture has changed something the
// window keeps.
package engines

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/roster"
	"github.com/e1i0r/orbit/internal/words"
)

// Knobs is the dials a run is made with: the engine, its model, how hard it
// thinks. The window keeps them — every screen that starts work reads them —
// and this screen is where they are turned.
type Knobs struct {
	Engine   string
	Model    string
	Effort   string
	Thinking string
}

// Env is what this screen needs of the world: the words it speaks, the keys
// it answers, the room it has, the catalogue of engines and what is left of
// each one's quota.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	Frame layout.Frame
	// Engines is the catalogue: every engine this build knows, whether this
	// machine can run it, and the dials each offers.
	Engines func() []roster.Engine
	// Quota is what is known about one engine's windows, drawn beside its
	// name because this is the screen where the choice is made.
	Quota func(engine string) roster.Reading
	// Settled is the engine a run uses when the dials name none: the
	// settings file's, or the first the catalogue lists.
	Settled string
}

// catalogue is every engine this build knows, and nothing at all in a
// window built without the door to ask. Saying nothing is the only honest
// answer this package has: it may not name internal/engine, so any table it
// carried would be a copy waiting to drift from the one the build runs.
func (e Env) catalogue() []roster.Engine {
	if e.Engines == nil {
		return nil
	}

	return e.Engines()
}

// dialled is the engine a name resolves to: the one named, or the one that
// is in force when nothing names another.
func (e Env) dialled(named string) string {
	if named != "" {
		return named
	}

	return e.Settled
}

// Out is what the screen asks the window for.
type Out struct {
	// Said is a sentence for the band.
	Said string
	// Leave is the reader closing the screen; Back is what they came from,
	// as the window's own number — which screens there are is the window's
	// business.
	Leave bool
	Back  int
	// Set is what to write to the settings file, in the order it was
	// turned. A dial the reader moved is a dial they expect to still be
	// there tomorrow.
	Set []Setting
}

// A Setting is one dial written down: what it is called in the settings
// file, and what it was turned to.
type Setting struct {
	Name  string
	Value string
}

// about names a value and what the sentence calls it.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}

// State is the screen: where the cursor is, which engines are showing their
// models, what has been typed to cut the list down, and the dials as the
// reader has left them.
type State struct {
	sel int

	// offset is the first line of the list on show, moved only by
	// keepEngineRowSeen: opencode alone offers sixty-four models, so this
	// list has been taller than the screen since the day it stopped being
	// a shortlist.
	offset int

	// open is which engines are showing their models. Absent is shut, so
	// the screen opens as four names and their dials. Nothing forces one
	// open or shut: two engines side by side is how a reader compares them.
	open map[string]bool

	// filter cuts the catalogue down to what was typed and typing is
	// whether the keyboard is going into it. See enginesfilter.go.
	filter string
	typing bool

	showingSetup bool
	setupEngine  string

	// knobs is the dials while the screen is up. It is seeded from the
	// window's at Open and handed back by Dials: the reader turning one
	// here is turning the window's.
	knobs Knobs

	// back is the screen this one was opened from, kept as the window's own
	// number and handed back when it leaves.
	back int
}

type engineRowKind int

const (
	rowHeader engineRowKind = iota
	rowEngine
	rowModel
	rowEffort
	rowThinking
)

type engineRow struct {
	kind     engineRowKind
	title    string
	engine   string
	id       string
	selected bool
	disabled bool
	open     bool
	models   int
	chosen   string
	setup    []string
}

// Open is the screen coming up, on the dials as they stand. back is the
// screen it was opened from, which leaving it returns to.
func Open(back int, dials Knobs) State {
	return State{knobs: dials, back: back}
}

// Dials is the knobs as the reader left them. The window keeps them: this
// screen is where they are turned and every screen that starts work reads
// them.
func (s State) Dials() Knobs { return s.knobs }

// View is the list drawn: every engine, and the models of the ones that are
// open.
func (s State) View(h, w int, e Env) []string {
	return s.rows(h, w, e)
}

// Hit is what the list has at that cell.
func (s State) Hit(x, y int, e Env) point.Target {
	return s.hit(x, y, e)
}

// Chip is the dials in one line, for the header, and nothing at all when
// none of them has been turned. It is a function of the dials and not of the
// screen: the header draws it while the screen is shut.
func Chip(k Knobs, e Env) string {
	if k.Engine == "" && k.Model == "" && k.Effort == "" && k.Thinking == "" {
		return ""
	}

	parts := []string{e.dialled(k.Engine)}

	if k.Model != "" {
		parts = append(parts, k.Model)
	}

	if k.Effort != "" {
		parts = append(parts, k.Effort)
	}

	// "off" is the default and says nothing; only a mode somebody turned on
	// is worth a word in a header this contested.
	if k.Thinking != "" && k.Thinking != "off" {
		parts = append(parts, "thinking")
	}

	return strings.Join(parts, cells.Dot)
}

// chosenModel is the model this engine would run with, for the row to carry
// while it is shut: folding hides which of sixty-five is in force, and the
// name of it is the one thing on that list a reader needs at a glance.
func (s State) chosenModel(eng roster.Engine, e Env) string {
	if eng.Name != e.dialled(s.knobs.Engine) || s.knobs.Model == "" {
		return ""
	}

	for _, mdl := range eng.Models {
		if mdl.ID == s.knobs.Model {
			return mdl.Label
		}
	}

	return s.knobs.Model
}

// selectableEngineIndices is which rows a reader can stand on: the headings
// are not among them.
func selectableEngineIndices(rows []engineRow) []int {
	var idxs []int

	for i, r := range rows {
		if r.kind != rowHeader {
			idxs = append(idxs, i)
		}
	}

	return idxs
}
