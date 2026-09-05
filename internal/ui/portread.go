package ui

// The two things the window reads through, declared here at the consumer.
//
// Both are interfaces rather than structs, and for one reason: internal/ui
// may not name internal/store, so it cannot take the concrete types. It says
// what it needs and something with the state root satisfies it — which is
// also what keeps the window from growing a second way to reach the record.
//
// They are in a file of their own because port.go answers a different
// question: what the window may be told to do, rather than what it may ask.

import (
	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
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
