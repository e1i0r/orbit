package verb

// The readings that are not folds of one task's record.
//
// Each answers twice, the way every reading here does: Saw is the thing
// itself, for a surface that draws structures, and Said is the same answer
// written out for a terminal. What they are made of comes through Sees,
// because a reading is not allowed to be the place a new opinion about the
// board appears.

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/quota"
)

// known is everything Orbit has been told about the code.
func known(w World) (Out, error) {
	facts, err := w.Facts()
	if err != nil {
		return Out{}, err
	}

	var b strings.Builder

	for _, f := range facts {
		// What it does and not what it asked to do: a fact that asked to
		// stop and brought no check warns, and a reader deciding whether to
		// trust it has to be told which they are looking at.
		does := "warns"
		if f.Action() == knowledge.Stops {
			does = "stops"
		}

		fmt.Fprintf(&b, "%-6s %-20s %s\n", does, scopeOf(f), f.Phrase)
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: facts}, nil
}

// scopeOf is what a fact is about, in one column.
//
// The columns are for a terminal; Saw carries the scope itself, so a surface
// that wants to draw it does not read this back.
func scopeOf(f knowledge.Fact) string {
	switch {
	case f.Scope.Symbol != "":
		return f.Scope.Path + ":" + f.Scope.Symbol
	case f.Scope.Path != "":
		return f.Scope.Path
	case f.Scope.Lang != "":
		return f.Scope.Lang
	case f.Scope.Repo != "":
		return filepath.Base(f.Scope.Repo)
	default:
		return "everywhere"
	}
}

// running is every engine this build knows, and what this machine has.
//
// The catalogue is the whole of it and not only what is installed: an engine
// missing from the answer is one nobody can be told how to install. What the
// telling looks like is a screen's business, and stays there.
func running() (Out, error) {
	all := knownEngines()

	var b strings.Builder

	for _, e := range all {
		here := "not installed"
		if e.Available {
			here = "installed"
		}

		fmt.Fprintf(&b, "%-12s %s\n", e.Name, here)

		// The models under the engine and not beside it. opencode answers
		// with sixty of them, and a line that long is one nobody reads to
		// the end of — which is the line that has the model they wanted.
		for _, line := range wrapped(e.Models, 72) {
			fmt.Fprintf(&b, "    %s\n", line)
		}
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: all}, nil
}

// wrapped is a list of names as lines no wider than a terminal.
//
// An empty name is dropped rather than printed as a gap: a dial whose first
// choice is "whatever the flow said" carries one, and a leading comma is how
// that read.
func wrapped(names []string, wide int) []string {
	var (
		out  []string
		line string
	)

	for _, n := range names {
		if n == "" {
			continue
		}

		switch {
		case line == "":
			line = n
		case len(line)+2+len(n) <= wide:
			line += ", " + n
		default:
			out = append(out, line+",")
			line = n
		}
	}

	if line != "" {
		out = append(out, line)
	}

	return out
}

// knownEngines is the catalogue, in the order a list of them is shown.
//
// Sorted, because a map has no order and a listing whose rows move between
// two readings is one nobody can learn the shape of. Where the program is
// comes from the engine and not from a guess that the name is the binary:
// opencode installs into ~/.opencode/bin and puts that on a shell profile,
// so a PATH exported before the install has no opencode in it.
func knownEngines() []Engine {
	all := engine.All()

	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}

	slices.Sort(names)

	out := make([]Engine, 0, len(names))

	for _, name := range names {
		one := all[name]
		if one == nil {
			continue
		}

		_, missing := one.Locate()

		out = append(out, Engine{
			Name:      name,
			Available: missing == nil,
			Models:    named(one.Models()),
			Efforts:   named(one.Efforts()),
			CanThink:  one.CanThink(),
		})
	}

	return out
}

// named is a dial's choices as they are typed.
func named(from []engine.Choice) []string {
	out := make([]string, 0, len(from))
	for _, c := range from {
		out = append(out, c.ID)
	}

	return out
}

// left is what is still in each engine's windows.
//
// An engine with no reading is listed with none rather than left out. "I do
// not know what is left" and "nothing is left" are different answers, and a
// queue that stopped is explained by one of them and not the other.
func left() (Out, error) {
	all := windows()

	var b strings.Builder

	for _, e := range all {
		if len(e.Windows) == 0 {
			fmt.Fprintf(&b, "%-12s no reading\n", e.Name)

			continue
		}

		for _, win := range e.Windows {
			fmt.Fprintf(&b, "%-12s %-10s %3.0f%% used, resets in %s\n",
				e.Name, win.Label, win.Pct, awhile(win.ResetsIn))
		}
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: all}, nil
}

// windows is each engine's quota, read without waiting for a sync.
//
// A reading asked for at a terminal that blocked on somebody's API would be
// a command that hangs, and the number it would come back with is the same
// one a moment later.
func windows() []Engine {
	m := quota.FromEnv()
	out := knownEngines()

	if m == nil {
		return out
	}

	for i := range out {
		one := m.Read(out[i].Name, false)
		out[i].Money, out[i].Sourced = one.Mode.Spends(), one.Sourced

		for _, w := range one.Windows {
			out[i].Windows = append(out[i].Windows, Window{
				Label: w.Label, Pct: w.Pct, ResetsIn: int(w.ResetsIn.Seconds()),
			})
		}
	}

	return out
}

// awhile is a count of seconds as a person says it.
func awhile(secs int) string {
	switch {
	case secs <= 0:
		return "now"
	case secs < 3600:
		return strconv.Itoa(secs/60) + "m"
	default:
		return strconv.Itoa(secs/3600) + "h" + strconv.Itoa(secs%3600/60) + "m"
	}
}
