package web

// What the screens beyond the board read through.
//
// Three of them are not readings of the record: what Orbit has learned, what
// has been said to the supervisor, and which engines this machine can run.
// Each lives behind a port for the reason the board does — this package
// knows no store, no engine and no state root, and the window is built the
// same way, so the two halves of Orbit read one set of facts through one set
// of adapters.
//
// The types are exported because a port only somebody inside this package
// can implement is not a port. They carry their own JSON tags rather than
// being copied into a second set of shapes on the way out: a copy of a copy
// is a place for the two to drift, and these are already the answer's shape
// rather than any package's internals.

import "time"

// Knows is everything Orbit has been told, across every repository.
type Knows interface {
	Facts() []Fact
}

// Fact is one thing Orbit has learned, as a reader reads it.
type Fact struct {
	Phrase string `json:"phrase"`
	// Scope is what it is about, already spelled: "repo ledger", "go", the
	// path of a file. The shape of a scope is internal/knowledge's, and a
	// page working it out from parts would be a second opinion about it.
	Scope string `json:"scope"`
	// Source is where it came from — read off the code, said by a person,
	// learned from a run.
	Source string `json:"source"`
	// Action is what it actually does, "stops" or "warns", and not what it
	// was asked to do: a fact that asked to stop and brought no check
	// warns, and a reader has to be told which they are looking at.
	Action string    `json:"action"`
	Check  string    `json:"check,omitempty"`
	Ref    string    `json:"ref,omitempty"`
	Repo   string    `json:"repo,omitempty"`
	At     time.Time `json:"at"`
	Used   int       `json:"used"`
	Off    bool      `json:"off,omitempty"`
}

// Talks is the supervisor's thread: the conversations, and every turn in
// them.
type Talks interface {
	Thread() ([]Chat, []Said, error)
}

// Chat is one conversation, as the thread lists it.
type Chat struct {
	ID    string    `json:"id"`
	Title string    `json:"title"`
	First time.Time `json:"first"`
	Last  time.Time `json:"last"`
	Turns int       `json:"turns"`
}

// Said is one turn: who said it, in which conversation, and about what.
type Said struct {
	At           time.Time `json:"at"`
	Kind         string    `json:"kind"`
	By           string    `json:"by,omitempty"`
	Channel      string    `json:"channel,omitempty"`
	Task         string    `json:"task,omitempty"`
	Repo         string    `json:"repo,omitempty"`
	Text         string    `json:"text,omitempty"`
	Conversation string    `json:"conversation,omitempty"`
}

// Roster is every engine this build knows and what this machine can do with
// them.
type Roster interface {
	Engines() []EngineInfo
	// Settled is the engine a run uses when nothing names one.
	Settled() string
}

// EngineInfo is one engine: whether it is here, what it can be turned to,
// and what is left of its windows.
type EngineInfo struct {
	Name      string   `json:"name"`
	Available bool     `json:"available"`
	Models    []string `json:"models,omitempty"`
	Efforts   []string `json:"efforts,omitempty"`
	CanThink  bool     `json:"canThink,omitempty"`
	// Setup is what a reader has to do to make it available, and is carried
	// only for an engine that is not: the screen shows it in place of the
	// dials.
	Setup []string `json:"setup,omitempty"`
	// Money says this engine is billed rather than rationed, and Sourced
	// that the reading below came from the engine itself rather than from a
	// guess. An unsourced reading is not shown as one.
	Money   bool     `json:"money,omitempty"`
	Sourced bool     `json:"sourced,omitempty"`
	Quota   []Window `json:"quota,omitempty"`
}

// Asks is the one thing a reader does anything through.
//
// One method and not a port per verb, because what the verbs are is not this
// package's to know: internal/verb declares them, once, for the command
// line, the window, the browser and the MCP server together. A port per verb
// was six methods that had to be added to in step with a list somewhere else
// — and the list they were meant to match had already drifted twice by the
// time this replaced them.
//
// A nil Asks is a server that shows no buttons. It is what a build without a
// store gets, and it is the honest state — not a set of controls that refuse
// when pressed.
type Asks interface {
	// Ask does one verb and answers what it said. A refusal is an error
	// with a sentence in it: "this task is already being run", "the board
	// is over its unread cap". Those are answers to the reader about the
	// state of their own machine, not a server that broke.
	Ask(name string, in Asked) (Answered, error)
}

// Asked is what a verb was asked with: the task if it is about one, and
// whatever the reader typed, by the names the verb takes.
type Asked struct {
	Task string
	Repo string
	Args map[string]string
}

// Answered is what it said.
type Answered struct {
	// Said is the sentence the reader is shown, written by the verb so
	// that the browser and the terminal report the same act in the same
	// words.
	Said string
	// Of is what it acted on, where that is a list worth showing.
	Of []string
	// Saw is what a reading read, in the shape it was read in.
	Saw any
}

// Told is everything ever said about a task, as markdown.
//
// A port and not a reading of the record here, because it is the same
// rendering the window draws and the same file an engine is handed when the
// terminal is opened on a task. Three copies of one conversation would be
// three chances for the browser to show something the terminal does not.
type Told interface {
	History(id, repo string) (string, error)
}

// Standings is the reading the buttons are chosen from.
//
// A port of its own and not a seventh verb, because it is not a verb: it
// changes nothing, it is read on every look at a task, and the task route
// asks for it where no button was pressed. One value fills both — it needs
// the same store and the same flow — and that is the point of them being
// two: a build could answer what a task can be asked for and refuse to do
// any of it.
type Standings interface {
	Standing(id, repo string) Standing
}

// Standing is what a task can be asked for right now.
type Standing struct {
	// Held is whether a process is running this task. It is not the band:
	// a run parked at a gate is in "needs you" and is very much alive, and
	// a page that chose its buttons off the band offered Start on a task
	// something was already running.
	Held bool `json:"held"`
	// Pending is what the dependency gate is waiting on somebody to
	// accept, and what makes Approve worth offering.
	Pending []string `json:"pending,omitempty"`
}

// Window is one quota window and how much of it is gone.
type Window struct {
	Label string  `json:"label"`
	Pct   float64 `json:"pct"`
	// ResetsIn is in seconds. A Go duration marshals as a count of
	// nanoseconds, which is a number no page should have to divide.
	ResetsIn int `json:"resetsIn"`
}
