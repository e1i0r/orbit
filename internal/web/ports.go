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

// Verbs is what a reader can do to the run of a task.
//
// Six words, and they are the whole of what steering a run means: begin one,
// hold it, let it go on, let it past the gate it stopped at, let it past the
// phase itself, and end it. None of them carries anything but the task —
// what a reader has to say goes through Says, next door.
//
// Every one of these changes what is happening on somebody's machine, and
// two of them spend money, so they arrive over POST and through the guard in
// verbs.go — never as a reading. Each takes the task by its id and the
// repository it is against, because that pair is what identifies a task
// everywhere in Orbit, and answers when the word is on disk rather than when
// the run has acted on it: whether it did is a question the record answers,
// and the page is already reading it.
//
// A nil Verbs is a server that shows no buttons. It is what a build without
// a store gets, and it is the honest state — not a set of controls that
// refuse when pressed.
type Verbs interface {
	// Start runs a task in a process of its own. It refuses a task
	// something else is already running, and a board over its unread cap.
	Start(id, repo string) error
	// Pause and Resume ask a run to stop and to carry on at its next phase
	// boundary. They are a pair: a browser that could pause and not resume
	// would be a way to strand a run nobody can reach from here.
	Pause(id, repo string) error
	Resume(id, repo string) error
	// Continue is resume's other half, and not the same word. Resume undoes
	// a pause somebody asked for; Continue lets a phase past the gate its
	// own flow asked it to stop at, which is what a run sitting in "needs
	// you" is waiting for. A browser with only Resume could not release
	// the one state a reader opens the page to release.
	Continue(id, repo string) error
	// Skip lets the run past the phase itself rather than past its gate.
	// Nothing is recorded for a phase that did not run.
	Skip(id, repo string) error
	// Cancel asks the run to stop where it stands, and to write down that
	// it was stopped.
	Cancel(id, repo string) error
}

// Says is what a reader can put on the record about a task, as against about
// its run.
//
// The line between this and Verbs is what each acts on. A word to a run only
// means something while one is walking; every one of these is recorded and
// waits for the next run if none is going — a note written on a task nobody
// is running is read by the phase that starts next, and that is the point of
// it. Three of the four carry the reader's own words.
type Says interface {
	// Note leaves a word for the phase that starts next.
	Note(id, repo, text string) error
	// Direct is a note the run is stopped for: the correction goes on the
	// record and the run in flight is asked to end, so the next one starts
	// having read it. restart begins that next one straight away, which
	// spends money.
	Direct(id, repo, text string, restart bool) error
	// Requeue takes a task back to the queue, stopping whatever holds it.
	// why is the reader's reason if they gave one.
	Requeue(id, repo, why string) error
	// Approve says yes to the libraries a task added, and answers which —
	// the reader is answering the question the record asked them, so what
	// was approved is what was pending, not a list from the caller.
	Approve(id, repo string) ([]string, error)
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
