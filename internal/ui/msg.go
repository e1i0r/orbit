package ui

// Every message the window has, and the command that raises each one, in a
// single file — so that "does every Cmd this program spawns have a Msg some
// Update handles?" is a question a reader answers by reading one screen
// rather than by grepping five.
//
// Nothing here reaches the state root. The window is handed the two things
// that can — a board.Reader that only ever reads, and a control function
// internal/cli builds over internal/task — and arch.layers leaves it no way
// to find a third: internal/record, internal/store and internal/engine are
// not on its list.

import (
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// The nine messages. Three of them are clocks, and they are separate
// clocks on purpose: the board is read twice a second, the tree is walked
// every two seconds because walking it is the expensive half, and the
// elapsed column is redrawn once a second because a column that only moved
// when something else did would look stopped.
type (
	// tickMsg is the poll: read the tail of every log that grew.
	tickMsg time.Time

	// rescanMsg is the enumeration: look for repositories and tasks that
	// appeared since the window opened.
	rescanMsg time.Time

	// boardMsg is a board that was read, and what moved since the last
	// one. A boardMsg whose Board has a zero ReadAt is a read that did not
	// happen — Update keeps the board it already has and says what went
	// wrong, because a failed stat must not blank a screen full of tasks.
	boardMsg struct {
		Board   board.Board
		Changed board.Changed
	}

	// elapsedMsg is the second hand, and it moves nothing but the clock
	// every row's age is measured against.
	elapsedMsg time.Time

	// diffMsg is the output of git diff for one task, or the reason there
	// is none. Text is carried whole rather than pre-wrapped: how wide the
	// pane is is not known to the process that ran git.
	//
	// Tree is where that diff was taken, and it comes back with the text
	// rather than being asked for a second time when o is pressed. The
	// window would otherwise have to reach the port from inside a
	// keystroke, which is a read off the event loop; and the path the
	// editor opens in has to be the path the diff was taken in, or the
	// file under the cursor is a file in some other directory.
	diffMsg struct {
		ID, Text, Tree string
		Err            error
	}

	// logMsg is one task's whole record, folded, or the reason there is
	// none. It carries the entries and not the events: internal/ui is not
	// allowed to know the record's own shape, and view.Entry is what it is
	// allowed to know instead.
	logMsg struct {
		ID      string
		Entries []view.Entry
		Err     error
	}

	// controlMsg is what the control function said when it was asked to
	// write one word. Err comes back verbatim — the window is a keyboard
	// in front of the commands, not a second copy of their rules.
	controlMsg struct {
		ID, Word string
		Err      error
	}

	// startedMsg is a run that began, and the process group it began in.
	// Nothing in this task raises one; the start dialog does.
	startedMsg struct {
		ID  string
		Pid int
		Err error
	}

	// editorMsg is $EDITOR having come back. There is nothing to say when
	// it went well — the screen is simply redrawn — so the message carries
	// only the failure.
	editorMsg struct{ Err error }

	// languageMsg is the reader having chosen a language. It is a message
	// rather than a direct call because changing language rebuilds the key
	// map, and a key map rebuilt outside the event loop is a key map the
	// frame on screen was not drawn from.
	languageMsg struct{ Lang string }
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

	// CanResume is whether the configured engine can resume a session,
	// which decides whether taking the keyboard is offered at all. It
	// arrives as a bool for the same reason Control arrives as a function:
	// internal/engine is not on internal/ui's import list, so the answer
	// has to be carried in rather than asked for. It is what fills
	// Conditions.CanResume, which task 10 declared and nothing yet set.
	CanResume bool
}

// tick asks for the next board poll.
func tick() tea.Cmd {
	return tea.Tick(board.RefreshEvery, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// rescanTick asks for the next enumeration.
func rescanTick() tea.Cmd {
	return tea.Tick(board.RescanEvery, func(t time.Time) tea.Msg { return rescanMsg(t) })
}

// elapsedTick asks for the next second.
func elapsedTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return elapsedMsg(t) })
}

// refresh reads the board once, off the event loop.
//
// A Reader that is nil is a window opened without one — a test of the
// rendering, or a frame drawn from a board handed straight in — and the
// honest answer is an empty message rather than a panic inside a Cmd, where
// the stack says nothing about which screen asked.
func refresh(r Reader) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return boardMsg{}
		}
		b, changed, err := r.Refresh()
		if err != nil {
			return boardMsg{Board: board.Board{Errs: []error{fmt.Errorf("read the board: %w", err)}}}
		}
		return boardMsg{Board: b, Changed: changed}
	}
}

// rescan walks the tree for what appeared since the window opened.
//
// Rescan answers with an error and nothing else: what it found shows up in
// the next refresh, not in this message. So the message it produces carries
// only what went wrong, which is the same shape a failed refresh has — one
// case in Update handles both, and the band is the one place a read failure
// is ever said.
func rescan(r Reader) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return boardMsg{}
		}
		if err := r.Rescan(); err != nil {
			return boardMsg{Board: board.Board{Errs: []error{fmt.Errorf("look for new repositories: %w", err)}}}
		}
		return boardMsg{}
	}
}

// control writes one word through the port and reports what it said.
//
// The error is passed on exactly as it arrived. The temptation these
// gestures have, and the one the window in the program this replaces did
// not, is that they are Go functions returning error rather than shell exit
// codes — so interpreting the failure looks free. It is not: an interpretation is a second copy of
// the command's rules, and the two copies disagree the first time the
// command changes its mind.
func control(port func(view.Task, string) error, t view.Task, word string) tea.Cmd {
	return func() tea.Msg {
		if port == nil {
			return controlMsg{ID: t.ID, Word: word, Err: errors.New("this window was opened without a way to control a task")}
		}
		return controlMsg{ID: t.ID, Word: word, Err: port(t, word)}
	}
}
