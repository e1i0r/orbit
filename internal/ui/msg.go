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
	"fmt"
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

// The twelve messages. Three of them are clocks, and they are separate
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
	//
	// NoBase is whether gitDiff fell back to a plain working-tree diff
	// because there was no base branch to compare against. Its zero value
	// is false — "there was a base" — on purpose: every diffMsg a test
	// builds by hand and never sets the field means the ordinary case, and
	// only a message from an actual fallback has to say otherwise.
	// Base is the branch this diff was measured against, and whether it
	// had been looked up when the diff was taken. It comes back so the
	// Model can hold it: the lookup costs three git subprocesses through a
	// helper that cannot be cancelled, and putting it on the rescan clock
	// was paying that every two seconds for an answer that does not change
	// while a view is open.
	diffMsg struct {
		ID, Text, Tree string
		Err            error
		NoBase         bool
		Base           baseRef
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

	// filesMsg is what a task's own directory holds, or the reason it
	// could not be read. It is a separate answer from the record because
	// it is a separate question: the events say what happened and the
	// directory says what is there, and neither can be derived from the
	// other.
	filesMsg struct {
		ID    string
		Files []view.File
		Err   error
	}

	// fileTextMsg is what one file of a task's directory holds, or the
	// reason it could not be read. It carries the name it was asked for,
	// because two of them can be out at once and the answers do not come
	// back in the order they were sent.
	fileTextMsg struct {
		ID, Name string
		Text     view.FileText
		Err      error
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

	// readMsg is a finished task having been marked read, or the reason it
	// was not. It is its own message rather than a controlMsg carrying the
	// word "read" because "read" is not one of the five words a run
	// understands: putting it on that wire would make the window the only
	// place in Orbit where the control vocabulary has six.
	readMsg struct {
		ID  string
		Err error
	}

	// requeuedMsg is a task having been taken back to the queue, or the
	// reason it was not.
	requeuedMsg struct {
		ID  string
		Err error
	}

	// sessionMsg is the interactive session t would open, as a command line
	// nobody has run yet.
	//
	// The command is built off the event loop and handed back rather than
	// started there, and the two halves are what make this gesture testable
	// at all: a test can read the arguments — that --fork-session is on
	// them — and stop, where a test of a gesture that ran the process would
	// open a session and spend money.
	sessionMsg struct {
		ID  string
		Cmd *exec.Cmd
		Err error
	}

	// sessionEndedMsg is the reader having come back from one. The window
	// was suspended for the whole of it, so this is also where the frame is
	// redrawn.
	sessionEndedMsg struct {
		ID  string
		Err error
	}

	// languageMsg is the reader having chosen a language. It is a message
	// rather than a direct call because changing language rebuilds the key
	// map, and a key map rebuilt outside the event loop is a key map the
	// frame on screen was not drawn from.
	languageMsg struct{ Lang string }

	// supervisorReplyMsg is the response the supervisor engine produced.
	supervisorReplyMsg struct {
		Text string
		Err  error
	}

	// spinnerTickMsg powers the live animated thinking indicator at 100ms.
	spinnerTickMsg time.Time
)

// spinnerTick asks for the next animation frame at 100ms.
func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg(t) })
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
			return boardMsg{Board: board.Board{Errs: []error{fmt.Errorf("%w: %w", errReadBoard, err)}}}
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
			return boardMsg{Board: board.Board{Errs: []error{fmt.Errorf("%w: %w", errFindRepos, err)}}}
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
			return controlMsg{ID: t.ID, Word: word, Err: errNoControlPort}
		}

		return controlMsg{ID: t.ID, Word: word, Err: port(t, word)}
	}
}

// start asks the port to begin a run, off the event loop.
//
// The refusal comes back exactly as task.Start phrased it. The window has
// already refused at the cap itself, with the ids of the tasks that are
// waiting, and this is the second look — taken by the function that owns the
// rule, against the settings file rather than against a board that may be
// half a second old. Two answers to one question, and the authoritative one
// is the one that is repeated verbatim.
func start(port func(view.Task, string, int) (int, error), t view.Task, flowName string, unread int) tea.Cmd {
	return func() tea.Msg {
		if port == nil {
			return startedMsg{ID: t.ID, Err: errNoStartPort}
		}

		pid, err := port(t, flowName, unread)

		return startedMsg{ID: t.ID, Pid: pid, Err: err}
	}
}

// markRead moves the brake back by one, through the port.
func markRead(port func(view.Task) error, t view.Task) tea.Cmd {
	return func() tea.Msg {
		if port == nil {
			return readMsg{ID: t.ID, Err: errNoReadPort}
		}

		return readMsg{ID: t.ID, Err: port(t)}
	}
}

// requeue takes a task back to the queue, through the port.
//
// It is off the event loop like every other gesture here, and it is the one
// that most needs to be: a run that has been signalled is waited on until
// its own process is gone, which is seconds at best and half a minute for an
// engine that has stopped listening.
func requeue(port func(view.Task) error, t view.Task) tea.Cmd {
	return func() tea.Msg {
		if port == nil {
			return requeuedMsg{ID: t.ID, Err: errNoRequeuePort}
		}

		return requeuedMsg{ID: t.ID, Err: port(t)}
	}
}

// takeSession builds the command line that carries on an engine's session,
// and builds it here rather than inside the keystroke because finding the
// session id means reading a record.
func takeSession(port func(view.Task) (*exec.Cmd, error), t view.Task) tea.Cmd {
	return func() tea.Msg {
		if port == nil {
			return sessionMsg{ID: t.ID, Err: errNoSessionPort}
		}

		cmd, err := port(t)

		return sessionMsg{ID: t.ID, Cmd: cmd, Err: err}
	}
}

// askSupervisorCmd queries the supervisor model in the background.
func askSupervisorCmd(ask func(string, string) (string, error), engineName, prompt string) tea.Cmd {
	return func() tea.Msg {
		if ask == nil {
			return supervisorReplyMsg{Err: errNoSupervisor}
		}

		ans, err := ask(engineName, prompt)

		return supervisorReplyMsg{Text: ans, Err: err}
	}
}
