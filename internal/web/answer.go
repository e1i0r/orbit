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

// taskAnswer is one task opened: the row, the record behind it, and the two
// readings of that record the panes are built on.
type taskAnswer struct {
	taskSummary
	Entries []entryAnswer `json:"entries"`
	// Walk is the files the task reached, in the order it first reached
	// them. It is worked out here rather than in the page because
	// internal/view owns the rule — which tool names mean a file changed
	// differs per engine, and a second reading of that would drift.
	Walk []stepAnswer `json:"walk"`
	// Spent is what the whole task has cost, summed off the phases.
	Spent float64 `json:"spent"`
	// Standing is what the task can be asked for right now. It travels
	// with the task rather than on a route of its own because it is what
	// decides which buttons are drawn, and a button that has to be pressed
	// to find out whether it does anything is a button nobody presses.
	Standing
}

// stepAnswer is one file the task touched.
type stepAnswer struct {
	Path    string `json:"path"`
	Touches int    `json:"touches"`
	Read    int    `json:"read"`
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
	// Story and Delta are what a phase said about its own work, when it
	// said anything: the shape of the change, and what it asks and
	// promises. They arrive on their own kinds and nowhere else.
	Story *storyAnswer `json:"story,omitempty"`
	Delta *deltaAnswer `json:"delta,omitempty"`
	// Truncated says the engine printed more than the record kept.
	Truncated bool `json:"truncated,omitempty"`
}

// storyAnswer is how a change came about, in the five parts the record keeps
// it in.
type storyAnswer struct {
	Entry   string `json:"entry,omitempty"`
	Purpose string `json:"purpose,omitempty"`
	Symptom string `json:"symptom,omitempty"`
	Cause   string `json:"cause,omitempty"`
	Fix     string `json:"fix,omitempty"`
}

// deltaAnswer is what the change asks of the code around it, and what it
// promises back.
type deltaAnswer struct {
	Needs      []string `json:"needs,omitempty"`
	Guarantees []string `json:"guarantees,omitempty"`
	Assumes    []string `json:"assumes,omitempty"`
	Instead    []string `json:"instead,omitempty"`
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

// fileAnswer is one file of the worktree, for opening a diff out past the
// context git wrote.
type fileAnswer struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Text    string `json:"text,omitempty"`
	Missing bool   `json:"missing,omitempty"`
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
func taskOf(t view.Task, entries []view.Entry, now Standing) taskAnswer {
	out := taskAnswer{taskSummary: summaryOf(t), Standing: now}

	for _, e := range entries {
		out.Spent += e.Cost

		out.Entries = append(out.Entries, entryAnswer{
			At: e.At, Kind: e.Kind, Phase: e.Phase, Attempt: e.Attempt,
			Text: said(e), Engine: e.Engine, Model: e.Model,
			Cost: e.Cost, Tool: e.Tool, Gate: e.Gate, Exit: e.Exit,
			Story: storyOf(e), Delta: deltaOf(e), Truncated: e.Truncated(),
		})
	}

	for _, step := range view.Walk(entries) {
		out.Walk = append(out.Walk, stepAnswer{
			Path: step.Path, Touches: step.Touches, Read: step.Read,
		})
	}

	return out
}

// storyOf is the story a phase told, and nothing for an entry that told
// none.
func storyOf(e view.Entry) *storyAnswer {
	s := e.Story
	if s == nil {
		return nil
	}

	return &storyAnswer{
		Entry: s.Entry, Purpose: s.Purpose, Symptom: s.Symptom,
		Cause: s.Cause, Fix: s.Fix,
	}
}

// deltaOf is what a phase said its change asks and promises.
func deltaOf(e view.Entry) *deltaAnswer {
	if e.Delta == nil || !e.Delta.Any() {
		return nil
	}

	return &deltaAnswer{
		Needs: e.Delta.Needs, Guarantees: e.Delta.Guarantees,
		Assumes: e.Delta.Assumes, Instead: e.Delta.Instead,
	}
}

// said is the line a reader reads.
//
// A tool call's text is the arguments the engine was given, as JSON. Sent as
// they arrived, the timeline is a wall of {"command":"grep -rn ..."} — the
// call is in there and nobody reads it. view.ToolLine is the rule that turns
// one into "grep: the thing it was looking for", and it is the same rule the
// band, the overview and the MCP server read tool calls by. Three renderings
// of one event would be three things to keep in step.
func said(e view.Entry) string {
	if line := view.ToolLine(e.Tool, e.Text); line != "" {
		return line
	}

	return e.Text
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
