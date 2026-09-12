package cli

// What a command is: what it takes, what it is for, what the window does
// when it is asked for one, and what it needs of the record in front of it.
//
// The commands themselves are the table in table.go. They are two files
// because this one met the size ceiling, and the seam is the honest one:
// this says what a command can be, and that says which ones there are.

import (
	"io"

	"github.com/e1i0r/orbit/internal/words"
)

// Context is what a command is handed besides its own arguments.
//
// The writers are here rather than reached for through os so that every
// command can be run in-process by a test, which is the same reason Run
// answers with an exit code instead of calling os.Exit. The printer is here
// so that it is built once per invocation: getting one means reading the
// settings file, and a command that reached for its own would open the state
// root to do it — which `orbit help` must not, and which nothing else needs
// to do twice.
type Context struct {
	Out   io.Writer
	Err   io.Writer
	Words *words.Printer
}

// InWindow is what the window does when it is asked for a command by name.
//
// It is on the command rather than in the window because it is a fact about
// the command: `run` blocks until a phase finishes, `top` is the window
// itself, and `list` is a question the window has already answered on
// screen. Keeping it here is what lets a command be added in one place.
type InWindow int

const (
	// WindowRuns is a command the window runs the way the command line
	// does, and prints the outcome of.
	WindowRuns InWindow = iota
	// WindowRefuses is a command that makes no sense from inside, and every
	// one of them carries the sentence saying why in Because. A refusal
	// that is a boolean is a refusal with nothing to print, so adding one
	// costs a sentence — which is the point.
	WindowRefuses
	// WindowOpens is a command the window answers with a screen rather than
	// with output: `list` is the board, `show` is the task view, `repos`
	// and `flows` are screens of their own. Printing a table into a pane
	// when the reader is already looking at the live version of it is the
	// answer to a question nobody asked.
	WindowOpens
)

// Command is one verb of the command line.
//
// About is a function of a printer and not a string because a description is
// something a reader reads: it goes through internal/words like every other
// sentence this program shows, and a literal here would be the one line of
// the interface that stayed in English.
type Command struct {
	Name  string
	Args  string // the usage fragment after the name, as a reader types it
	About func(*words.Printer) string
	Run   func(Context, []string) error

	InWindow InWindow
	Because  func(*words.Printer) string // why, when InWindow is WindowRefuses

	// Salvage says the command still runs when the record is too damaged
	// for the migration in front of it to finish. Two commands are: `check`,
	// which says what is wrong with it, and `export`, which gets out
	// whatever can still be read. Everything else stops, because a state
	// root half moved is the one shape nobody can reason about — but
	// stopping these two would mean a file that breaks takes the only two
	// commands for a broken file down with it.
	Salvage bool

	// OffRecord says the command neither reads nor writes the record, so the
	// record being unusable is none of its business and the three
	// maintenance steps in front of every command are skipped for it.
	// Three commands set it: `version`, which prints a constant this binary
	// was built with; `upgrade`, which talks to GitHub and to `go install`
	// and to nothing else; and `repos`, which walks a directory tree looking
	// for git repositories.
	//
	// It is a different fact from Salvage and not a stronger one. A salvage
	// command opens the record and tolerates what it finds; an off-record
	// one never opens it, so there is nothing to tolerate.
	//
	// The reason it exists is a dead end rather than a convenience: the
	// refusal a newer record gives says to run `orbit upgrade`, and that
	// command was stopped by the refusal it was the way out of.
	OffRecord bool

	// NeedsArgs says the command refuses when it is given none — the id of a
	// task, for most of them, and a directory for `export`. Every command
	// that sets it has an Args fragment saying what it wants.
	//
	// It is a fact about the command and not about any one entry point,
	// which is why the board's menu can read it: that menu chooses with no
	// arguments at all, so an entry for one of these ran bare and came back
	// with the refusal, which reads as an entry that is broken rather than
	// as one in the wrong place.
	NeedsArgs bool

	// AboutATask says the argument it wants first is the id of a task, and
	// it implies NeedsArgs — an id is an argument. The board's menu leaves
	// these out altogether: it is the menu of no row in particular, and a
	// verb about one task belongs to the menu of the task it is about.
	AboutATask bool
}

// printer is the Context's own, and English for a Context that was built
// without one. Nothing in the program builds one that way — Run always asks
// language() — but a Context is three plain fields, so a caller writing one
// by hand gets a usage screen rather than a nil dereference in the middle of
// printing one.
func (c Context) printer() *words.Printer {
	if c.Words == nil {
		return words.For("")
	}

	return c.Words
}

// Usage is the command as the usage screen prints it.
func (c Command) Usage() string {
	if c.Args == "" {
		return "orbit " + c.Name
	}

	return "orbit " + c.Name + " " + c.Args
}
