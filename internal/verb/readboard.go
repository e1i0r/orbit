package verb

// The readings that are about the board rather than about one task: what is
// on it, what one row's record says, the shapes work can walk, and the
// checkouts it is walked in.

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// listed is every task under this root, in the band that says what it waits
// for.
func listed(w World) (Out, error) {
	b, err := w.Board()
	if err != nil {
		return Out{}, err
	}

	var out strings.Builder

	for _, t := range b.Tasks {
		fmt.Fprintf(&out, "%-14s %-12s %-10s %s\n", t.ID, view.BandOf(t), t.Repo, t.Title)
	}

	if len(b.Tasks) == 0 {
		return Out{Said: "no tasks under this root", Saw: b.Tasks}, nil
	}

	return Out{Said: strings.TrimRight(out.String(), "\n"), Saw: b.Tasks}, nil
}

// shown is everything the record says about one task.
func shown(w World, in In) (Out, error) {
	b, err := w.Board()
	if err != nil {
		return Out{}, err
	}

	row, found := rowOf(b.Tasks, in.Task)
	if !found {
		return Out{}, fmt.Errorf("no task %q on the board", in.Task)
	}

	entries, err := w.Log(row.RepoPath, row.ID)
	if err != nil {
		return Out{}, err
	}

	var out strings.Builder

	fmt.Fprintf(&out, "%s  %s  %s\n", row.ID, view.BandOf(row), row.Title)

	for _, e := range entries {
		fmt.Fprintf(&out, "%s  %-18s %s\n", e.At.Format("15:04:05"), e.Kind, oneLine(e.Text))
	}

	return Out{Said: strings.TrimRight(out.String(), "\n"), Saw: entries}, nil
}

// rowOf is the task one id means on the board.
func rowOf(tasks []view.Task, id string) (view.Task, bool) {
	for _, t := range tasks {
		if t.ID == id {
			return t, true
		}
	}

	return view.Task{}, false
}

// oneLine is an event's text as a row of a listing: the first line of it,
// because an engine's answer runs to paragraphs and this is a column.
func oneLine(text string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(text), "\n")

	return first
}

// shapes is every flow a task can be started under.
func shapes(w World) (Out, error) {
	all := flow.List(w.Store())

	var out strings.Builder

	p := w.Words()
	for _, one := range all {
		fmt.Fprintf(&out, "%-16s %s\n", one.Name, FlowMark(p, one.Origin))
	}

	return Out{Said: strings.TrimRight(out.String(), "\n"), Saw: all}, nil
}

// FlowMark is where a flow came from, in the reader's own language.
//
// The classification is flow.List's and the sentence is this one's, which is
// the split that lets a translation test see the words: a mark spliced in as
// a Go constant inside internal/flow was invisible to both the honesty test
// and the pseudolocale golden. It is exported because `orbit flows` and the
// window's start dialog draw the same three facts, and two spellings of
// "yours, shadowing the built-in" is the drift this is here to stop.
//
// OriginUnknown reaches here only if List ever returned it. It does not, and
// the empty string is what an unmarkable name gets rather than a panic in a
// listing.
func FlowMark(p *words.Printer, o flow.Origin) string {
	switch o {
	case flow.OriginBuiltin:
		return p.T("flow.built_in", "built in")
	case flow.OriginUser:
		return p.T("flow.yours", "yours")
	case flow.OriginShadow:
		return p.T("flow.shadowing", "yours, shadowing the built-in")
	}

	return ""
}

// checkouts is every repository Orbit is watching, and the work in each.
func checkouts(w World) (Out, error) {
	b, err := w.Board()
	if err != nil {
		return Out{}, err
	}

	held := map[string]int{}
	for _, t := range b.Tasks {
		held[t.Repo]++
	}

	var out strings.Builder

	for _, r := range b.RepoList {
		fmt.Fprintf(&out, "%-20s %3d  %s\n", r.Name, held[r.Name], r.Path)
	}

	if len(b.RepoList) == 0 {
		return Out{Said: "no repositories under this root", Saw: b.RepoList}, nil
	}

	return Out{Said: strings.TrimRight(out.String(), "\n"), Saw: b.RepoList}, nil
}

// written puts the record back out as JSON lines.
func written(w World, in In) (Out, error) {
	into := in.Arg("into")
	if !filepath.IsAbs(into) {
		// Relative to wherever the caller stands, which is what a person
		// typing `orbit export ./out` means. Making it absolute here is
		// what stops two ways in disagreeing about where "out" is.
		abs, err := filepath.Abs(into)
		if err != nil {
			return Out{}, err
		}

		into = abs
	}

	said, err := w.Export(into, in.Task)
	if err != nil {
		return Out{}, err
	}

	return Out{Said: said, Of: []string{into}}, nil
}

// handed gives a terminal to an engine, in the task's own checkout.
func handed(w World, in In) (Out, error) {
	said, err := w.Take(in.Task, in.Repo)
	if err != nil {
		return Out{}, err
	}

	return Out{Said: said}, nil
}
