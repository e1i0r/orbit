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

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/flow"
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
	Engine() string
	Model() string
	Flow() string
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
}

// Options is everything the window is handed. Every field is a value or a
// port; none of them is a path, and none of them is a handle on the state
// root.
type Options struct {
	Root     string // where the repositories are, for the header and the empty state
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

	// Take builds the interactive session t suspends the window for, and
	// does not run it. It comes back as a command line so that the window
	// hands it straight to tea.ExecProcess and never has to know what
	// engine it is, where the session id came from, or where the worktree
	// is — three facts internal/ui is not allowed to reach.
	Take func(t view.Task) (*exec.Cmd, error)

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
}

// Command is one row of the palette: what the window shows of a command,
// and nothing of what the command does.
//
// About and Because are functions of a printer rather than strings because
// both are sentences a reader reads, and sentences go through
// internal/words like every other line this window draws — which also lets
// them follow a language changed after this slice was handed over.
type Command struct {
	Name  string // as the reader types it
	Args  string // the usage fragment after the name; empty when none
	About func(*words.Printer) string

	Refused bool                        // the window does not run it here
	Because func(*words.Printer) string // why, when Refused is set
}
