// Package repos is the repository list: every repository the board found,
// with its name, its path, and how many tasks of each band it holds.
// Choosing one filters the board to it.
//
// It is entered through this file. State is which row the cursor is on, Env
// is the board and the words the window lends it, and Out is what it asks
// for when the reader chooses.
package repos

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// Env is what this screen needs of the world: the words it speaks, the keys
// it answers, the board it lists, and which repository the board is already
// filtered to.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	// Board is what was found: the repositories walked, and the tasks that
	// name them.
	Board board.Board
	// Filter is the repository the board is showing, and empty when it is
	// showing all of them.
	Filter string
}

// Out is what the screen asks the window for.
type Out struct {
	// Said is a sentence for the band.
	Said string
	// Leave is the reader closing the screen.
	Leave bool
	// Filter is the repository to show, and Cleared says to show them all —
	// the two are different answers, and an empty string alone cannot tell
	// them apart.
	Filter  string
	Cleared bool
}

// State is the screen: which row the cursor is on.
type State struct {
	sel int
}

// about names a value and what the sentence calls it.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}

// An Item is one repository as this screen lists it: what it is called,
// where the checkout is, and how many tasks of each band name it.
//
// The window reads these too — a click asks which repository a row is, and
// the form that starts a task asks where one lives — so the list is a door
// as well as a drawing.
type Item struct {
	Name   string
	Path   string
	counts [4]int
	total  int
}

// Open is the screen coming up, on the first repository.
func Open() State { return State{} }

// List is every repository the board found, which the window also draws a
// count of in its header.
func List(e Env) []Item { return collect(e) }

func collect(e Env) []Item {
	var list []Item

	seen := make(map[string]int)

	if len(e.Board.RepoList) > 0 {
		for _, r := range e.Board.RepoList {
			idx := len(list)
			seen[strings.ToLower(r.Name)] = idx
			list = append(list, Item{Name: r.Name, Path: r.Path})
		}
	}

	// A task counts in every repository it was worked in. This screen is a
	// list of repositories and not of tasks, so a task that reached into
	// three of them is three repositories with work going on in them — the
	// one place in the window where counting the pairs is the true answer.
	for _, t := range e.Board.Tasks {
		for _, name := range worked(t) {
			idx, ok := seen[strings.ToLower(name)]
			if !ok {
				idx = len(list)
				seen[strings.ToLower(name)] = idx
				list = append(list, Item{Name: name, Path: pathOf(t, name)})
			}

			band := view.BandOf(t)
			if band >= 0 && int(band) < len(list[idx].counts) {
				list[idx].counts[band]++
				list[idx].total++
			}
		}
	}

	return list
}

// worked is the repositories of one row, and the single name a row carries
// when it carries no list.
func worked(t view.Task) []string {
	if len(t.Repos) > 0 {
		return t.Repos
	}

	return []string{t.Repo}
}

// pathOf is where a repository of a task is, which is known only for the one
// the task is filed under: the row carries the others by name. A repository
// the board walked has its path in RepoList already, and this answer is only
// reached by one it did not.
func pathOf(t view.Task, name string) string {
	if strings.EqualFold(name, t.Repo) {
		return t.RepoPath
	}

	return ""
}

// Key is every key on this screen.
func (s State) Key(msg tea.KeyPressMsg, e Env) (State, Out) {
	repos := collect(e)
	if len(repos) == 0 {
		if key.Matches(msg, e.Keys.Back) || key.Matches(msg, e.Keys.Quit) {
			return State{}, Out{Leave: true}
		}

		return s, Out{}
	}

	switch {
	case key.Matches(msg, e.Keys.Back) || key.Matches(msg, e.Keys.Quit):
		return State{}, Out{Leave: true}
	case key.Matches(msg, e.Keys.Up):
		s.sel--
		if s.sel < 0 {
			s.sel = len(repos) - 1
		}

		return s, Out{}
	case key.Matches(msg, e.Keys.Down):
		s.sel++
		if s.sel >= len(repos) {
			s.sel = 0
		}

		return s, Out{}
	case key.Matches(msg, e.Keys.Open), msg.Text == " ":
		return s.chose(repos[s.sel], e)
	}

	return s, Out{}
}

// Choose takes the repository of that name, which is what a click on one
// does — on this screen or on the header's chip.
func (s State) Choose(name string, e Env) (State, Out) {
	for i, r := range collect(e) {
		if strings.EqualFold(r.Name, name) {
			s.sel = i

			return s.chose(r, e)
		}
	}

	return s, Out{}
}

// chose is what choosing one repository answers.
//
// Choosing the one already showing is how the filter is taken off: the same
// gesture both ways, so a reader who filtered by pressing ⏎ does not have to
// find another key to undo it.
func (s State) chose(chosen Item, e Env) (State, Out) {
	p := e.Words

	if strings.EqualFold(e.Filter, chosen.Name) {
		return State{}, Out{
			Leave:   true,
			Cleared: true,
			Said:    p.T("repos.filter_cleared", "showing all repositories"),
		}
	}

	return State{}, Out{
		Leave:  true,
		Filter: chosen.Name,
		Said:   p.T("repos.filtered", "filtered to {repo}", about("repo", chosen.Name)),
	}
}

// View is the list drawn.
func (s State) View(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	p := e.Words
	out := []string{
		"",
		"  " + theme.Paint(theme.Accent).Render(p.T("repos.title", "Repositories")),
		"  " + theme.Paint(theme.Dim).Render(p.T("repos.subtitle", "choose a repository to filter the board")),
		"",
	}

	repos := collect(e)
	if len(repos) == 0 {
		out = append(out, "  "+theme.Paint(theme.Dim).Render(p.T("repos.none", "no repositories found")))
	}

	for i, r := range repos {
		mark := strings.Repeat(" ", cells.Gutter)
		if i == s.sel {
			mark = cells.Mark + strings.Repeat(" ", cells.Gutter-1)
		}

		nameRendered := theme.Paint(theme.Accent).Render(r.Name)
		if strings.EqualFold(e.Filter, r.Name) {
			nameRendered += " " + theme.Paint(theme.OK).Render(p.T("repos.active", "[filtered]"))
		}

		countsStr := p.T("repos.counts", "{todo} to do · {running} in flight · {needs} needs you · {done} done",
			about("todo", strconv.Itoa(r.counts[0])),
			about("running", strconv.Itoa(r.counts[1])),
			about("needs", strconv.Itoa(r.counts[2])),
			about("done", strconv.Itoa(r.counts[3])))

		line := fmt.Sprintf("%s%-24s  %s  (%s)",
			mark,
			nameRendered,
			theme.Paint(theme.Dim).Render(r.Path),
			theme.Paint(theme.Dim).Render(countsStr),
		)
		out = append(out, cells.Fit(line, w))
	}

	waysOut := p.T("repos.ways_out", "{open} filter · {up_down} move · {back} back",
		about("open", e.Keys.Open.Help().Key),
		about("up_down", e.Keys.Up.Help().Key+e.Keys.Down.Help().Key),
		about("back", e.Keys.Back.Help().Key))
	out = append(out, "", cells.Fit("  "+theme.Paint(theme.Dim).Render(waysOut), w))

	return cells.Fill(out, h)
}
