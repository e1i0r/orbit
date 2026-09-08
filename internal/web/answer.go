package web

// What the routes answer with.
//
// These are the shapes the page reads, and they are written down here rather
// than marshalled straight from internal/view for one reason: view's structs
// are the fold of the record and they change when the record does. A page
// served from a browser tab somebody left open yesterday should not break
// because a field was renamed inside the program.

import (
	"time"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

// boardAnswer is the whole board: the rows and where they are.
type boardAnswer struct {
	Root   string        `json:"root"`
	Repos  []repoAnswer  `json:"repos"`
	Bands  []bandAnswer  `json:"bands"`
	Tasks  []taskSummary `json:"tasks"`
	ReadAt time.Time     `json:"readAt"`
}

type repoAnswer struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// bandAnswer is one band and how many are in it, in the order the window
// draws them.
type bandAnswer struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// taskSummary is a row: what the board shows without opening anything.
type taskSummary struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Band   string   `json:"band"`
	Repo   string   `json:"repo"`
	Repos  []string `json:"repos"`
	Flow   string   `json:"flow"`
	Phase  string   `json:"phase"`
	Engine string   `json:"engine"`
	Model  string   `json:"model"`
}

// taskAnswer is one task opened: the row, and the record behind it.
type taskAnswer struct {
	taskSummary
	Entries []entryAnswer `json:"entries"`
}

// entryAnswer is one line of the record. It carries what a reader needs to
// see what happened and nothing about how it is drawn.
type entryAnswer struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	Phase   string    `json:"phase"`
	Attempt int       `json:"attempt"`
	Text    string    `json:"text,omitempty"`
	Engine  string    `json:"engine,omitempty"`
	Model   string    `json:"model,omitempty"`
	Cost    float64   `json:"cost,omitempty"`
	Tool    string    `json:"tool,omitempty"`
	Gate    string    `json:"gate,omitempty"`
	Exit    string    `json:"exit,omitempty"`
}

// diffAnswer is what a task changed.
//
// Empty and Missing are different facts and the page says different things
// about them: a task that changed nothing, and a task whose checkout is not
// there to look at.
type diffAnswer struct {
	ID      string `json:"id"`
	Text    string `json:"text,omitempty"`
	Empty   bool   `json:"empty,omitempty"`
	Missing bool   `json:"missing,omitempty"`
	Failed  string `json:"failed,omitempty"`
}

// boardOf turns a board into what the page reads.
func boardOf(b board.Board, root string) boardAnswer {
	out := boardAnswer{Root: root, ReadAt: b.ReadAt}

	for _, r := range b.RepoList {
		out.Repos = append(out.Repos, repoAnswer{Name: r.Name, Path: r.Path})
	}

	for _, band := range view.Bands() {
		out.Bands = append(out.Bands, bandAnswer{Name: bandName(band), Count: b.Counts[band]})
	}

	for _, t := range b.Tasks {
		out.Tasks = append(out.Tasks, summaryOf(t))
	}

	return out
}

// taskOf is one task with its record behind it.
func taskOf(t view.Task, entries []view.Entry) taskAnswer {
	out := taskAnswer{taskSummary: summaryOf(t)}

	for _, e := range entries {
		out.Entries = append(out.Entries, entryAnswer{
			At: e.At, Kind: e.Kind, Phase: e.Phase, Attempt: e.Attempt,
			Text: e.Text, Engine: e.Engine, Model: e.Model,
			Cost: e.Cost, Tool: e.Tool, Gate: e.Gate, Exit: e.Exit,
		})
	}

	return out
}

func summaryOf(t view.Task) taskSummary {
	return taskSummary{
		ID: t.ID, Title: t.Title, Band: bandName(view.BandOf(t)),
		Repo: t.Repo, Repos: t.Repos, Flow: t.Flow,
		Phase: t.Phase, Engine: t.Engine, Model: t.Model,
	}
}

// bandName is the band as the page names it: a word, not a number, because
// the numbers are internal/view's to renumber.
func bandName(b view.Band) string {
	switch b {
	case view.NeedsYou:
		return "needs_you"
	case view.Running:
		return "running"
	case view.ToDo:
		return "todo"
	case view.Done:
		return "done"
	}

	return "unknown"
}
