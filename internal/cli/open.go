package cli

// The interactive session the cockpit's `c` hands the terminal to.
//
// It is built here rather than in the window for the reason ports.go gives —
// the window cannot name a store, and it may not name internal/mcp either —
// and the arrangement is worth more here than anywhere else: the session
// this file builds is one that can call back into Orbit while it runs. The
// window decides which task the reader is looking at; this decides what a
// model is told about it and what it is allowed to reach.

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/mcp"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
)

// openPort builds the session the window suspends itself for, and runs
// nothing.
//
// dir is the window's own answer to "where" — the repository the cursor is
// on, or the first on the board when it is on nothing — and it is refined
// here rather than replaced: only this side can ask where a task's worktree
// is.
func openPort(s *store.Store, r *board.Reader) func(t view.Task, engineName, dir string) (*exec.Cmd, error) {
	return func(t view.Task, engineName, dir string) (*exec.Cmd, error) {
		cmd, err := openCommand(engineName, openDir(r, t, dir), openContext(t))
		if err != nil {
			return nil, err
		}

		openJournal(s, t, engineName)

		return cmd, nil
	}
}

// openJournal writes down that the terminal was handed to a session on this
// task, so that the record says an hour of work happened here.
//
// Without it the session is the one thing that changes a task and leaves no
// trace: a run writes phases, a note writes a note, and this writes into a
// worktree while the record shows the same failed attempt it showed before.
// The dialogue tab is where the reader goes to ask "what has been said about
// this", and a session they opened themselves belongs in that answer.
//
// It is written when the session is built rather than when it ends, because
// the process it comes back from is one the window suspends itself for: a
// terminal that never returns — the reader closes the window, the machine
// sleeps — would take the only evidence the session happened with it.
//
// A failure here is logged and nothing more. The reader pressed a key to be
// handed a terminal; a record that could not be appended to is not a reason
// to refuse them one.
func openJournal(s *store.Store, t view.Task, engineName string) {
	if s == nil || t.ID == "" {
		return
	}

	if err := task.Dialogue(s, subject(t), engineName, "the cockpit handed the terminal to an interactive session on this task"); err != nil {
		logger.Error("cli/open", "record the session opened on %s: %v", t.ID, err)
	}
}

// openDir is where the session opens: the task's own worktree when there is
// one on disk, and what the window suggested otherwise.
//
// The worktree is the checkout a run made its changes in, and a session
// opened in the repository would be reading a tree that does not have them.
// A task that has never run has no worktree, and a cursor on a band header
// has no task at all; both are ordinary, and both fall back rather than
// fail.
func openDir(r *board.Reader, t view.Task, fallback string) string {
	if r == nil || t.ID == "" || t.RepoPath == "" {
		return fallback
	}

	worktree, err := r.Worktree(t.RepoPath, t.ID)
	if err != nil {
		return fallback
	}

	if info, err := os.Stat(worktree); err != nil || !info.IsDir() {
		return fallback
	}

	return worktree
}

// openCommand is the command line itself: the engine the reader chose,
// Orbit's own MCP server configured for this one session, and the task as
// the first thing said.
//
// The configuration is passed on the command line rather than written into
// the engine's own, which is the difference between this and `orbit mcp
// install`. Pressing a key in a window is not a request that every future
// session of that engine, in every directory, gain a server; it is a request
// about this one.
func openCommand(engineName, dir, context string) (*exec.Cmd, error) {
	if engineName == "" {
		return nil, fmt.Errorf("opening an interactive session needs an engine")
	}

	var args []string

	if flag := mcpConfigFlag(engineName); flag != "" {
		config, err := mcp.LaunchConfig("")
		if err != nil {
			return nil, err
		}

		args = append(args, flag, config)
	}

	if context != "" {
		// The separator first, when a flag was given. --mcp-config takes as
		// many values as follow it, so the sentence the session opens on was
		// read as a second configuration file: claude answered "MCP config
		// file not found: I am looking at orbit task ..." and exited before
		// it drew anything, which from the cockpit is a screen that flashes
		// and comes back.
		if len(args) > 0 {
			args = append(args, "--")
		}

		args = append(args, context)
	}

	cmd := exec.Command(engineName, args...)
	cmd.Dir = dir

	return cmd, nil
}

// mcpConfigFlag is how one engine is told about a server for the length of a
// session, and "" for an engine that has no such flag.
//
// It is a name check rather than a method on engine.Engine because the
// engine here is a word off a knob: the reader can put a name in it that
// this build does not run, and a session is still opened — the terminal is
// handed over, and the program either exists or says it does not. An engine
// that takes no configuration flag simply gets none, and the session is the
// one it would have had before this file existed.
func mcpConfigFlag(engineName string) string {
	if engineName == "claude" {
		return "--mcp-config"
	}

	return ""
}

// openContext is what the session is told before the reader types anything.
//
// It names the task and says the tools are there, and stops. Anything more
// would be this file deciding what the reader wants done, which is the one
// thing pressing `c` says they are about to decide themselves.
func openContext(t view.Task) string {
	if t.ID == "" {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "I am looking at orbit task %s", t.ID)

	if t.Repo != "" {
		fmt.Fprintf(&b, " in %s", t.Repo)
	}

	if t.Title != "" {
		fmt.Fprintf(&b, ": %s", t.Title)
	}

	fmt.Fprintf(&b, ". It is %s", t.Band)

	if t.Phase != "" {
		fmt.Fprintf(&b, ", in the %s phase", t.Phase)
	}

	b.WriteString(". Orbit's own mcp server is configured in this session: orbit_inspect_task reads this task's record — its notes, its gates and the last thing that went wrong — and orbit_add_note writes back to it, where the cockpit will show it. Read the record before changing anything.")

	return b.String()
}
