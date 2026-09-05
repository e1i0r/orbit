package ui

// The ports: everything the window is handed instead of a state root.
//
// It is a file of its own because it answers a different question from
// msg.go's. That file is "what can happen"; this one is "what may this
// window reach", and the answer is four interfaces' worth of reading and
// four closures — no path, no store, no engine, nothing that appends.
//
// Every one of them is declared here, at the consumer, which is this
// repository's rule for a port: *board.Reader and *store.Store satisfy them
// without naming them, so the packages that do the work owe nothing to the
// package that draws.

import (
	"io"
	"os/exec"
	"time"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// Settings is the window's port to the settings file: the standing answers
// it reads and the two it writes.
//
// It has five methods, where this repository's convention is one to three,
// and that is deliberate. The alternative — an interface per answer — would
// be five ports to one file, and a caller satisfying five ports with one
// type learns nothing the compiler did not already know. It is a port to
// one thing, and the thing has five questions.
//
// It is an interface and not a struct because internal/ui cannot name
// store.Settings: the window is not allowed to know where the settings
// live, only how to ask. internal/cli satisfies it over the store.
type Settings interface {
	Autopilot() bool
	SetAutopilot(bool) error
	Language() string
	SetLanguage(string) error
	UnreadCap() int
	SetUnreadCap(int) error
	// BudgetWorkspace and QuotaFloor are read and not written here: they
	// are the two halves of the brake the queue asks about, and `orbit set`
	// is where a number like that is chosen. A pair of writers this window
	// has no screen for would be a port promising a gesture nobody can
	// make.
	BudgetWorkspace() float64
	QuotaFloor() int
	Engine() string
	SetEngine(string) error
	Model() string
	SetModel(string) error
	Flow() string
	SetFlow(string) error
	Theme() string
	SetTheme(string) error
}

// Reader is the window's port to the state root, and everything it may ask
// of it is on this list.
//
// It is an interface for the reason Settings is: internal/ui cannot name the
// types the answers are made of. The board's own two methods would not have
// needed one — board.Board is on the allowed list and *board.Reader could
// have stayed a concrete type — but the two the task view adds do. A task's
// record is []record.Event and its worktree is a path only internal/store
// can compute, and neither package is on internal/ui's import list. The
// alternative was to widen that list, which would have let the window reach
// the writer as well as the reader; a port of four methods is the cheaper
// half of that trade.
//
// It is declared here, at the consumer, as every other port in this
// repository is: *board.Reader satisfies it without naming it, so the
// package that does the reading owes nothing to the package that draws.
type Reader interface {
	// Refresh is the poll: the board as it is now, and what moved.
	Refresh() (board.Board, board.Changed, error)
	// Rescan is the enumeration: look for repositories and tasks that
	// appeared since the window opened.
	Rescan() error
	// Log is one task's whole record, folded into entries.
	Log(repoPath, id string) ([]view.Entry, error)
	// Worktree is where that task's throwaway checkout lives.
	Worktree(repoPath, id string) (string, error)
	// Files is what that task's own directory holds, with each file's
	// size. The artifacts tab lists what a run left behind, and the only
	// way to list it honestly is to look.
	Files(repoPath, id string) ([]view.File, error)
	// FileText is what one of those files holds, from the start and up to
	// whatever the port is willing to read. It is asked for one file at a
	// time, when a reader opens it.
	FileText(repoPath, id, name string) (view.FileText, error)
	// SupervisorLog is the cockpit's persistent supervisor conversation thread.
	SupervisorLog() ([]view.SupervisorLine, error)
}

// Delivery is one press of a delivery key on its way into a task's record:
// what was asked for, what was handed the work, and — once it comes back —
// what it did or why it broke.
//
// It is one struct rather than five arguments because the two halves are one
// fact in two moments. A verb that is asked for and never answered is the
// shape of a key that appears to do nothing, and that is exactly what the
// reader has to be able to see: the ask on its own, until the answer joins
// it.
type Delivery struct {
	Verb    string // the caption the key was offered under, and never the command underneath
	By      string // what was handed the work: the supervisor, or the command that carries it
	Text    string // what came back, on the answer
	Failure error  // why it broke, where it did
	Done    bool   // whether this is the answer rather than the ask
}

// Options is everything the window is handed. Every field is a value or a
// port; none of them is a path, and none of them is a handle on the state
// root.
type Options struct {
	Root string // where the repositories are, for the header and the empty state
	// Version is the running build's own version, and it is here for one
	// question: whether the release GitHub reports is news. internal/ui
	// cannot name internal/cli, where the version is stamped in, so it
	// arrives the way every other fact from outside does.
	Version  string
	Reader   Reader
	Settings Settings
	Words    *words.Printer
	Width    int // 0 unless the caller is rendering one frame with --once
	Height   int

	// Control is how a gesture reaches the function its subcommand calls.
	// It is a port rather than a direct call to task.Control because every
	// entry point in internal/task takes a *store.Store, and internal/ui
	// cannot name that type — which is the arrangement arch.layers exists
	// to hold, not a gap in it. internal/cli, where the store and the
	// window are allowed to meet, supplies the closure.
	Control func(t view.Task, word string) error

	// Start is how the dialog begins a run, and it is the first caller
	// task.Start has ever had: the unread cap it enforces was dead code
	// until a window pressed it. unread is passed rather than worked out on
	// the other side because internal/task must not fold the record a
	// second time — the window is already holding a folded board, and
	// board.Unread of the board on screen is the number the reader can see.
	//
	// It answers with the pid the run was given, which is the only thing a
	// window can say about a process it does not own.
	Start func(t view.Task, flowName string, unread int) (int, error)

	// MarkRead is what moves that same brake back: the gesture behind d,
	// and the only thing on this screen that decrements unread 3/5.
	MarkRead func(t view.Task) error

	// Requeue stops whatever holds a task and puts it back in to do. It is
	// a port of its own rather than a sixth control word because the word
	// channel is read by a run between phases, and this gesture has to work
	// on a task with no run to read it.
	Requeue func(t view.Task) error

	// RecordSupervisor records a message into the global supervisor conversation thread.
	RecordSupervisor func(by, channel, message string) error

	// Learn writes down something the operator said about the code rather
	// than about a task: /rule for a fact that stops the work at a gate,
	// /aware for one that only reaches the prompt. scope is empty for the
	// repository named by repo, "general" for everything, or a language.
	//
	// It is a port and not a call because the window writes nothing: it says
	// what was meant and something else does it, which is the same line
	// every other gesture on this screen is drawn on.
	Learn func(stops bool, scope, repo, phrase string) error

	// Knows is what Orbit has learned about the code being worked in: the
	// general facts, the ones of every language, and the repository's own —
	// the same set a phase started there would be told.
	//
	// It is a port because reading them means reaching the state root and the
	// checkout, which the window may not do. What comes back is data.
	Knows func() []knowledge.Fact

	// KnowsAll is everything Orbit has learned, across the repositories on
	// the board and the state root, for the screen that lists it whole.
	KnowsAll func() []knowledge.Fact

	// TurnFact writes a fact back with whatever was changed about it, which
	// today is only whether it is switched off.
	TurnFact func(f knowledge.Fact) error

	// NoteTask puts a line in one task's notes, which is where a mention of
	// it in the supervisor lands.
	NoteTask func(id, text string) error

	// RetractSupervisor takes back one turn of that thread, named by the
	// moment it was written — an event carries no id, so its timestamp is
	// the only thing that already tells one line from another. Nothing is
	// erased: the line stays in the thread, marked, and stops being put in
	// front of the model.
	RetractSupervisor func(at time.Time) error

	// RecordDeliver writes one of the delivery keys onto the task it was
	// pressed about: the ask when the key goes down, and the answer when it
	// lands. A nil port writes nothing, which is what every one of those
	// keys did before there was one.
	//
	// It is a port and not a call to internal/task for the reason Control
	// is, and it is the same trip Delivery's own doc argues for: the window
	// says which task and what was asked of it, and the side that may append
	// decides how that is written down.
	RecordDeliver func(t view.Task, d Delivery) error

	// AskSupervisor asks the active engine to process and reply to the supervisor thread.
	AskSupervisor func(engineName, prompt string) (string, error)

	// AutoSupervise triggers the supervisor autonomously for tasks needing attention.
	AutoSupervise func(engineName string, taskIDs []string) (string, error)

	// DeleteTask permanently deletes a task record and its worktree from the store.
	DeleteTask func(t view.Task) error

	// Take builds the interactive session t suspends the window for, and
	// does not run it. It comes back as a command line so that the window
	// hands it straight to tea.ExecProcess and never has to know what
	// engine it is, where the session id came from, or where the worktree
	// is — three facts internal/ui is not allowed to reach.
	Take func(t view.Task) (*exec.Cmd, error)

	// Open is the session the c gesture hands the terminal to: a plain
	// interactive engine rather than a run, opened on a task rather than on
	// nothing.
	//
	// It is a port for the same reason Take is, and for one more. The
	// session it builds is handed Orbit's own MCP server, so the window
	// would otherwise have to name internal/mcp — a package that can write
	// to the record through internal/task, which is precisely the authority
	// arch.layers keeps out of here. The window says which task and where
	// it thinks the session belongs; the other side decides what the
	// session can reach.
	//
	// A window given no port opens the engine on its own, which is what it
	// did before there was one.
	Open func(t view.Task, engineName, dir string) (*exec.Cmd, error)

	// FileSession takes what was said in one of those sessions into the
	// task it was opened on, and answers how many turns it wrote.
	//
	// The window is suspended for the length of a session and sees none of
	// it, so the one thing it can say afterwards is when it handed the
	// terminal over. The other side reads the engine's own transcript for
	// that worktree and writes each turn into the record, which is the
	// only place this window can then show it. Tool calls are not turns:
	// what a session did to the files is the timeline's business, and this
	// is the conversation, which is the half that exists nowhere else once
	// the terminal is closed.
	//
	// A nil port files nothing, which is what a session left behind before
	// anything read one back.
	FileSession func(t view.Task, engineName string, since time.Time) (int, error)

	// Flows is where a user's own flows live, for the cycle the start
	// dialog offers. A nil Source is the built-ins and nothing else, which
	// is what flow.Resolve already documents and what a window opened
	// without a state root gets.
	Flows flow.Source

	// CanResume is whether the engine a task ran under can carry on a
	// session it started before, which decides whether taking the keyboard
	// is offered for that task at all. It is a function for the same reason
	// Control is: internal/engine is not on internal/ui's import list, so
	// the answer has to be carried in rather than asked for.
	//
	// It takes the engine's name rather than being a standing bool because
	// the sentence it produces names one engine. A program-wide answer was
	// an AND over every engine configured, so two engines of which one
	// could not resume refused every task — and told each of them that its
	// own engine was the one that could not. The engine's name comes off
	// the task, which is the only place that fact is recorded.
	//
	// A nil port answers no. A window handed no way to ask knows nothing
	// about engines, and a key that is offered and then refused is worse
	// than one greyed out with its reason.
	CanResume func(engine string) bool

	// Engines returns the engines the UI offers dials and setup steps for.
	Engines func() []EngineInfo

	// Commands is the command table the palette shows, carried in rather
	// than reached for: internal/cli owns the table and internal/ui cannot
	// import it, so the window is handed only what a reader must *see* —
	// the name, the usage fragment, the description, and whether the
	// window refuses the command here, with its reason.
	//
	// It deliberately does not carry Run. The window names a command; it
	// does not call one — that is what Do is for, and the split is what
	// keeps internal/ui from becoming a second copy of any command's rules.
	Commands []Command

	// Do runs one named command with its arguments, off in internal/cli
	// where the table lives, writing whatever the command prints to out.
	// It is how `orbit new` gets written from the window without
	// internal/ui importing anything that appends. A nil port answers
	// nothing: a window opened without one has no palette that can run,
	// and says so if asked.
	//
	// out is what makes the run watchable: the window keeps reading it
	// while the command runs, so a reader sees the work and not only the
	// verdict. Elio asked for that outright on 2026-08-23 — see the work
	// as it happens, then the result — and it overrides the band-only
	// sentence the plan first described.
	Do func(name string, args []string, out io.Writer) error

	// ValidID is the id rule the compose form types against, satisfied by
	// the store's own validator — the same one `orbit new` hits on its way
	// through Create. Two copies of an id rule is how an id the window
	// accepts arrives at a store that refuses it, which is the failure the
	// port exists to prevent. A nil port validates nothing: a window
	// opened without one submits the id and lets the command's own answer
	// come back verbatim.
	ValidID func(id string) error

	// Quota asks what is left of one engine's allowance, and in what unit
	// that engine's use is spoken about at all. A nil port answers nothing,
	// and the status line falls back on money — which is what every
	// engine's own command line reports.
	//
	// It takes an engine name because both answers are per engine: claude
	// under a subscription has a window and no dollars, codex on an API key
	// has dollars and no window, and a board can hold tasks of both.
	Quota func(engine string) QuotaReading
}
