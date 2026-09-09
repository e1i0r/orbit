package verb

// What the flow's own checks say on both sides of the change.
//
// The other three readings of an impact are folds of things already written
// down: the history, the test names, the engine's account. This one is not a
// reading at all. It checks out the base, runs somebody's test suite twice
// and costs minutes — so it is a verb, asked for on purpose, and never
// something a page refresh can set going.
//
// What it answers is the question the other three cannot: not "what might
// this reach" but "what did it actually break, and what did it fix".

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/repo"
)

// Sides is one check, run on the base and on the work.
type Sides struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	// Base and Now are what each side answered. A side that would not run
	// at all says so in Failed rather than reporting an exit nobody got.
	Base Side `json:"base"`
	Now  Side `json:"now"`
	// Broke is a check the change turned from passing to failing, and Fixed
	// the other way round. Both false is a check that answered alike, or one
	// that could not be run — which the reader is told apart by Failed.
	Broke bool `json:"broke,omitempty"`
	Fixed bool `json:"fixed,omitempty"`
}

// Side is what one command answered on one side of the change.
type Side struct {
	Exit   int    `json:"exit"`
	Out    string `json:"out,omitempty"`
	Failed string `json:"failed,omitempty"`
}

// weighed runs the flow's checks on both sides of the task's change.
func weighed(w World, in In) (Out, error) {
	t, dir, err := checkout(w, in)
	if err != nil {
		return Out{}, err
	}

	if dir == "" {
		return Out{}, fmt.Errorf("%s has no checkout to run anything in", t.ID)
	}

	chosen := t.Flow
	if chosen == "" {
		chosen = flow.Default
	}

	f, err := flow.Resolve(w.Store(), chosen)
	if err != nil {
		return Out{}, err
	}

	checks := checksOf(f)
	if len(checks) == 0 {
		return Out{Said: t.ID + "'s flow has no checks, so there is nothing to run on either side"}, nil
	}

	one, err := repo.Open(t.Repo.Path)
	if err != nil {
		return Out{}, err
	}

	got, err := one.Compare(dir, checks)
	if err != nil {
		return Out{}, err
	}

	sides := bothSides(got)

	return Out{Said: apart(sides), Saw: sides}, nil
}

// checksOf is every check a flow stops on: its phases' gates, and the ones
// its loops go round until.
func checksOf(f flow.Flow) []repo.Check {
	var out []repo.Check

	for _, p := range f.Phases {
		out = append(out, checksIn(p)...)
	}

	return out
}

// checksIn is one phase's checks, and its loop's.
func checksIn(p flow.Phase) []repo.Check {
	var out []repo.Check

	for _, g := range p.Gates {
		out = append(out, repo.Check{Name: g.Name, Command: g.Command})
	}

	if p.Loop == nil {
		return out
	}

	for _, g := range p.Loop.Until {
		out = append(out, repo.Check{Name: g.Name, Command: g.Command})
	}

	for _, inner := range p.Loop.Phases {
		out = append(out, checksIn(inner)...)
	}

	return out
}

// bothSides is the comparison in the shape a surface draws.
func bothSides(all []repo.Divergence) []Sides {
	out := make([]Sides, 0, len(all))

	for _, d := range all {
		one := Sides{
			Name:    d.Name,
			Command: d.Command,
			Base:    sideOf(d.Base),
			Now:     sideOf(d.Now),
		}

		// Broke and Fixed are only claimed where both sides actually ran.
		// A check that could not run on one side is not a regression, and
		// calling it one is how a reader learns to distrust the whole
		// section.
		if d.Base.Failed == nil && d.Now.Failed == nil {
			one.Broke = d.Base.Exit == 0 && d.Now.Exit != 0
			one.Fixed = d.Base.Exit != 0 && d.Now.Exit == 0
		}

		out = append(out, one)
	}

	return out
}

// sideOf carries one side across.
func sideOf(r repo.Ran) Side {
	out := Side{Exit: r.Exit, Out: r.Out}
	if r.Failed != nil {
		out.Failed = r.Failed.Error()
	}

	return out
}

// apart is the comparison as a terminal reads it.
func apart(sides []Sides) string {
	var b strings.Builder

	for _, s := range sides {
		mark := "same"

		switch {
		case s.Broke:
			mark = "BROKE"
		case s.Fixed:
			mark = "fixed"
		case s.Base.Failed != "" || s.Now.Failed != "":
			mark = "did not run"
		}

		fmt.Fprintf(&b, "%-12s %s\n", mark, s.Name)
	}

	return strings.TrimRight(b.String(), "\n")
}
