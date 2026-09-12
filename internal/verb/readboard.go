package verb

// The readings that are about the board rather than about one task: what is
// on it, what one row's record says, the shapes work can walk, and the
// checkouts it is walked in.

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// listed is every task the state root holds, in the band that says what
// it waits for — or every task worked in the repository named, when one
// was. Unnamed means everything rather than wherever the shell happens to
// be standing: a workspace holds tasks that reach into three repositories
// and tasks that reach into none at all, and a listing that answered the
// current directory would answer two different questions from two
// directories with nobody having asked differently.
func listed(w World, in In) (Out, error) {
	var ids []string

	if at := in.Arg("repo"); at != "" {
		one, err := repo.Open(at)
		if err != nil {
			return Out{}, err
		}

		ids, err = task.List(w.Store(), one)
		if err != nil {
			return Out{}, err
		}
	} else {
		var err error

		ids, err = w.Store().TaskIDs()
		if err != nil {
			return Out{}, err
		}
	}

	rows := make([]view.Task, 0, len(ids))

	for _, id := range ids {
		row, found, err := rowAnywhere(w, id)
		if err != nil {
			return Out{}, err
		}

		if found {
			rows = append(rows, row)
		}
	}

	var out strings.Builder

	for _, t := range rows {
		fmt.Fprintf(&out, "%-14s %-12s %-10s %s\n", t.ID, view.BandOf(t), t.Repo, t.Title)
	}

	if len(rows) == 0 {
		return Out{Said: "no tasks yet", Saw: rows}, nil
	}

	return Out{Said: strings.TrimRight(out.String(), "\n"), Saw: rows}, nil
}

// rowAnywhere is the row one id means, wherever it was worked. The board
// only carries what its root can see; a door with no root — a command run
// from anywhere — reads the record instead.
func rowAnywhere(w World, id string) (view.Task, bool, error) {
	events, err := task.Events(w.Store(), task.Task{ID: id})
	if err != nil {
		return view.Task{}, false, err
	}

	if len(events) == 0 {
		return view.Task{}, false, nil
	}

	row := view.Fold(events)
	row.ID = id

	paths, err := w.Store().TaskRepos(id)
	if err != nil {
		return view.Task{}, false, err
	}

	if len(paths) > 0 {
		row.RepoPath = paths[0]
		row.Repo = filepath.Base(paths[0])
	}

	return row, true, nil
}

// shown is everything the record says about one task.
func shown(w World, in In) (Out, error) {
	row, found, err := rowAnywhere(w, in.Task)
	if err != nil {
		return Out{}, err
	}

	if !found {
		return Out{}, fmt.Errorf("%s", w.Words().T("show.nothing_recorded", "nothing recorded for {id}",
			words.Arg{Name: "id", Value: in.Task}))
	}

	entries, err := w.Log(row.RepoPath, row.ID)
	if err != nil {
		return Out{}, err
	}

	var out strings.Builder

	fmt.Fprintf(&out, "%s  %s  %s\n", row.ID, view.BandOf(row), row.Title)

	for _, e := range entries {
		fmt.Fprintf(&out, "%s  %-18s %s\n", stamp(e.At), e.Kind, firstLine(detailOf(e)))
	}

	return Out{Said: strings.TrimRight(out.String(), "\n"), Saw: entries}, nil
}

// stamp says when, with the day included: a task that ran last week printed
// as 15:04:05 reads as though it ran this morning.
//
// An entry with no time is not a task that happened at the start of the
// Christian era — it is the placeholder for a line the record could not
// parse. Printing 0001-01-01 there would be a date, and a wrong one.
func stamp(t time.Time) string {
	if t.IsZero() {
		return "—"
	}

	return t.Format("2006-01-02 15:04:05")
}

// detailOf is what the last cell says about an entry, and for one that
// ended badly that is why it ended.
//
// Text is what the engine printed. The reason a phase failed is folded into
// Cause, so that a row ending at phase.failed still says why — and a row
// that reads "phase.failed" and then quotes the last line the engine
// happened to print reads as though that line were the failure. So the
// reason wins the cell when there is one; what the engine printed is in the
// log, which is a file you can read.
func detailOf(e view.Entry) string {
	if e.Cause != "" {
		return e.Cause
	}

	if e.Text == "" {
		// A repository joining is the one entry whose whole content is in
		// its fields: the name is what the row is about, and without it
		// the line reads "repo.joined" over an empty cell.
		return e.Repo
	}

	return e.Text
}

// firstLine keeps the table a table. The whole text stays in the record,
// which is a file you can read with cat.
//
// The text is whatever the engine printed, and this table is tab-delimited:
// one tab inside it silently adds a column and every row below drifts, and a
// carriage return would redraw the row over itself on a terminal.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i] + " …"
	}

	return strings.NewReplacer("\t", " ", "\r", " ").Replace(s)
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
