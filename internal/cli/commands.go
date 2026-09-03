package cli

// The interface as a table: every command orbit has, what it takes, what it
// is for, and what the window does when it is asked for one.
//
// One table, read by everything. The dispatcher looks a name up in it, the
// usage screen is printed from it, and the window's command line reaches
// commands through it rather than through a second list of names. What this
// replaces was a switch in Run beside a hand-written synopsis beside a
// hand-aligned help text — three copies of one list, kept in agreement only
// by somebody remembering to change all three, which is how a command ended
// up dispatchable under a name the usage screen never printed.

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

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

// commands is every command there is, in the order they are worth reading:
// the window first, then what a task is made to do, then what it is asked
// about, and the settings last.
func commands() []Command {
	return []Command{{
		Name: "top", Args: "[dir]",
		About:    func(p *words.Printer) string { return p.T("cmd.top", "watch every task in one window") },
		Run:      top,
		InWindow: WindowRefuses,
		Because:  func(p *words.Printer) string { return p.T("cmd.top.inside", "you are already in it") },
	}, {
		Name: "repos", Args: "[dir]",
		About:    func(p *words.Printer) string { return p.T("cmd.repos", "list the repositories under a directory") },
		Run:      repos,
		InWindow: WindowOpens,
	}, {
		Name:     "flows",
		About:    func(p *words.Printer) string { return p.T("cmd.flows", "list the flows a task can be written against") },
		Run:      flows,
		InWindow: WindowOpens,
	}, {
		Name: "new", Args: "-repo <dir> -id <id> <text>",
		About: func(p *words.Printer) string { return p.T("cmd.new", "write a task down") },
		Run:   newTask,
	}, {
		Name: "run", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string { return p.T("cmd.run", "run a task through its flow") },
		Run:   runTask,
	}, {
		Name: "pause", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string { return p.T("cmd.pause", "stop a run at its next phase") },
		Run:   func(ctx Context, args []string) error { return controlTask("pause", ctx, args) },
	}, {
		Name: "resume", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string { return p.T("cmd.resume", "let a stopped run carry on") },
		Run:   func(ctx Context, args []string) error { return controlTask("resume", ctx, args) },
	}, {
		Name: "list", Args: "-repo <dir>",
		About:    func(p *words.Printer) string { return p.T("cmd.list", "list the tasks of a repository") },
		Run:      list,
		InWindow: WindowOpens,
	}, {
		Name: "show", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About:    func(p *words.Printer) string { return p.T("cmd.show", "print what happened to a task") },
		Run:      show,
		InWindow: WindowOpens,
	}, {
		Name: "read", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string { return p.T("cmd.read", "mark a finished task as looked at") },
		Run:   readTask,
	}, {
		Name: "pr", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.pr", "create a pull request from a task's worktree")
		},
		Run: createPR,
	}, {
		Name: "merge", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.merge", "merge a task's pull request and delete its branch")
		},
		Run: mergePR,
	}, {
		Name: "close-pr", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.close_pr", "close a task's pull request on GitHub")
		},
		Run: closePR,
	}, {
		Name: "cancel", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string { return p.T("cmd.cancel", "stop a run, and say so in its record") },
		Run:   cancelTask,
	}, {
		Name: "requeue", Args: "-repo <dir> <id> [why]", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.requeue", "stop a run and put the task back in to do")
		},
		Run: requeueTask,
	}, {
		Name: "approve", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.approve", "say yes to the libraries a task added, so its next run goes past the gate")
		},
		Run: approveTask,
	}, {
		Name: "join", Args: "[-repo <dir>] [-task <id>] <name>", NeedsArgs: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.join", "open a checkout of another repository for a task")
		},
		Run: joinRepo,
	}, {
		Name: "reconcile", Args: "-repo <dir> [id]",
		About: func(p *words.Printer) string {
			return p.T("cmd.reconcile", "close the records of runs whose processes are gone")
		},
		Run: reconcile,
	}, {
		Name: "settings", Args: "[key] [value]",
		About:    func(p *words.Printer) string { return p.T("cmd.settings", "view or change settings") },
		Run:      set,
		InWindow: WindowOpens,
	}, {
		Name: "direct", Args: "-repo <dir> [-restart] <id> <message>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.direct", "interrupt or redirect a task and record the directive")
		},
		Run: directTask,
	}, {
		Name: "note", Args: "-repo <dir> <id> <text>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string { return p.T("cmd.note", "record a note for a task") },
		Run:   noteTask,
	}, {
		Name: "export", Args: "[-task <id>] <dir>", NeedsArgs: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.export", "write the record back out as JSONL, one file per task")
		},
		Run:     exportRecord,
		Salvage: true,
	}, {
		Name:  "check",
		About: func(p *words.Printer) string { return p.T("cmd.check", "say whether the record is still readable") },
		Run:   checkCommand,

		Salvage: true,
	}, {
		Name:  "version",
		About: func(p *words.Printer) string { return p.T("cmd.version", "print the version orbit was built at") },
		Run:   version,
	}, {
		Name:  "upgrade",
		About: func(p *words.Printer) string { return p.T("cmd.upgrade", "check for updates and upgrade orbit") },
		Run:   upgrade,
	}, {
		Name: "supervisor", Args: "[-by <author>] [-retract <n>] [text]",
		About: func(p *words.Printer) string {
			return p.T("cmd.supervisor", "read or write to the persistent supervisor conversation thread")
		},
		Run: supervisorCommand,
	}, {
		Name: "mcp", Args: "[install] [-root <dir>]",
		About: func(p *words.Printer) string {
			return p.T("cmd.mcp", "run the model context protocol server, or register it in the clients that speak it")
		},
		Run: runMCP,
		// The server owns this process's standard input and output for as
		// long as it runs, and the window owns the terminal those are
		// attached to. Running it from inside would hand the client the
		// window's screen and the window the client's requests.
		InWindow: WindowRefuses,
		Because: func(p *words.Printer) string {
			return p.T("cmd.mcp.inside", "it speaks over this terminal, which the window is already using")
		},
	}}
}

// lookup finds a command by the name that was typed.
func lookup(name string) (Command, bool) {
	for _, c := range commands() {
		if c.Name == name {
			return c, true
		}
	}
	//nolint:misspell // Spanish aliases for settings command
	if name == "set" || name == "config" || name == "configuracion" || name == "configuraciones" {
		for _, c := range commands() {
			if c.Name == "settings" {
				return c, true
			}
		}
	}

	return Command{}, false
}

// usage is the whole of orbit on one screen.
//
// The columns are laid out by tabwriter rather than by hand, because they
// have to line up on their own: the `new` line is longer than the rest, the
// hand-counted spaces that once aligned them stopped aligning the moment it
// was written, and a translated description is a width nobody can count in
// advance at all.
func usage(p *words.Printer) string {
	var b strings.Builder
	b.WriteString(p.T("cli.tagline", "orbit — a cockpit for supervising coding agents") + "\n\n")

	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	for _, c := range commands() {
		fmt.Fprintf(w, "  %s\t%s\n", c.Usage(), c.About(p))
	}

	_ = w.Flush() // a strings.Builder cannot fail to be written to

	b.WriteString("\n" + p.T("cli.state", "State lives in $ORBIT_HOME, or ~/.orbit when that is unset.") + "\n")

	return b.String()
}
