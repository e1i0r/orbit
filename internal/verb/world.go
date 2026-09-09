package verb

// What a verb needs of the machine it runs on.
//
// Ports, and small ones. The same verb has to work for a command line that
// opened a store a moment ago, a server holding one open for hours, and a
// test with neither — so what a verb reaches for is declared here and filled
// by whoever is asking.
//
// They are grouped by what they are about rather than by verb: a port per
// verb would be this package's own list written twice, which is the thing it
// exists to stop.

import (
	"context"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// World is everything the verbs reach for: the machine they run on, and
// through Beyond, the places they can reach past it.
//
// Two interfaces composed rather than one long one, and the seam is not
// arbitrary: everything in Here is Orbit's own state, and everything in
// Beyond costs money or is visible to somebody else.
type World interface {
	Here
	Beyond
	Speaks
	Sees
}

// Speaks is the reader's own language.
//
// A verb writes the sentence a reader is shown — that is the point of Said,
// and why the browser and the terminal report one act in one form of words.
// Which language those words are in is the surface's, because it is the
// surface that knows who is reading.
type Speaks interface {
	Words() *words.Printer
}

// Here is Orbit's own state on this machine.
type Here interface {
	// Store is where a task is written down and read back.
	Store() *store.Store
	// Find is the task one id in one repository means. It is a port rather
	// than task.Load called directly because opening a repository is the
	// caller's business: a command line has a -repo flag, a server has a
	// board that already walked them, and a test has a fixture.
	Find(id, repoPath string) (task.Task, repo.Repo, error)
	// Unread is how many finished tasks nobody has looked at, which is the
	// one number task.Start will not work out for itself.
	Unread(repoPath string) (int, error)
	// Looked tells the board a task was written or removed, so the next
	// read of it finds one that was not there before.
	Looked() error
}

// Sees is what a reading needs and cannot fold for itself.
//
// Three, and all three for the same reason: which repositories are in view
// is the caller's, not this package's. A command line has a -repo flag and a
// root it was pointed at; a server has a board it has held open for hours; a
// tool call has whatever the client's session names.
type Sees interface {
	// Facts is everything Orbit has been told, across those repositories.
	Facts() ([]knowledge.Fact, error)
	// Board is every task in them, in the band that says what it waits for.
	Board() (board.Board, error)
	// Log is one task's record, folded by the package that owns the format.
	// A second fold here would be a second opinion about what happened.
	Log(repoPath, id string) ([]view.Entry, error)
}

// Engine is one engine: whether it is here, what it can be turned to, and
// what is left of its windows.
//
// Its own shape and not the roster's, because the roster's carries closures
// that draw for a terminal, and a reading is answered to four surfaces.
type Engine struct {
	Name      string   `json:"name"`
	Available bool     `json:"available"`
	Models    []string `json:"models,omitempty"`
	Efforts   []string `json:"efforts,omitempty"`
	CanThink  bool     `json:"canThink,omitempty"`
	Setup     []string `json:"setup,omitempty"`
	// Money says this engine is billed rather than rationed, and Sourced
	// that the reading came from the engine itself rather than a guess.
	Money   bool     `json:"money,omitempty"`
	Sourced bool     `json:"sourced,omitempty"`
	Windows []Window `json:"windows,omitempty"`
}

// Window is one quota window and how much of it is gone.
type Window struct {
	Label string  `json:"label"`
	Pct   float64 `json:"pct"`
	// ResetsIn is in seconds. A Go duration marshals as a count of
	// nanoseconds, which is a number no surface should have to divide.
	ResetsIn int `json:"resetsIn"`
}

// Beyond is what a verb can reach past this machine: a model that has to be
// paid for, and a pull request other people will see.
type Beyond interface {
	// Deliver hands a task's work to the world — a pull request opened,
	// merged or closed. It is a port because what those mean is a page of
	// decisions about gh, remotes and branches, and they live where they
	// are rather than being copied here.
	Deliver(ctx context.Context, t task.Task, verb string) (string, error)
	// Say puts something in the supervisor's thread. It records and does
	// not answer: what the supervisor makes of it is its own loop's, and a
	// verb that quietly called a model would be one that spent money
	// without saying so.
	Say(text, by, about string) error
	// Learn writes down something true about the code.
	Learn(fact knowledge.Fact) error
	// Export writes the record back out as JSON lines, one file per task,
	// into a directory that must not already hold anything. A port because
	// where a way in is allowed to write files is the way in's business.
	Export(into, only string) (string, error)
	// Take hands a terminal to an engine in the task's own checkout. It is
	// the one verb that needs a terminal, so a way in that has none says so
	// rather than pretending.
	Take(id, repoPath string) (string, error)
}
