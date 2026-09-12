package cli

// The interface as a table: every command orbit has, in one list.
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
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/e1i0r/orbit/internal/words"
)

// commands is every command there is, in the order they are worth reading:
// the window first, then what a task is made to do, then what it is asked
// about, the settings, and last the three words of pause.go and the two
// verbs of answering.go — which are there because this file met the size
// ceiling.
func commands() []Command {
	return withVerbs(slices.Concat([]Command{{
		Name: "top", Args: "[dir]",
		About:    func(p *words.Printer) string { return p.T("cmd.top", "watch every task in one window") },
		Run:      top,
		InWindow: WindowRefuses,
		Because:  func(p *words.Printer) string { return p.T("cmd.top.inside", "you are already in it") },
	}, {
		Name: "web", Args: "[dir]",
		About:    func(p *words.Printer) string { return p.T("cmd.web", "read the same board in a browser") },
		Run:      serveWeb,
		InWindow: WindowRefuses,
		Because:  func(p *words.Printer) string { return p.T("cmd.web.inside", "it would serve the window you are in") },
	}, {
		Name: "repos", Args: "[dir]",
		About:    func(p *words.Printer) string { return p.T("cmd.repos", "list the repositories under a directory") },
		Run:      repos,
		InWindow: WindowOpens,

		// It walks a directory tree and reports the git repositories in it.
		// What Orbit has recorded about them is a different answer, and this
		// one is asked before there is a record to ask it of.
		OffRecord: true,
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
		Name: "history", Args: "[-repo <dir>] [-write] <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.history", "print everything ever said about a task, in any program")
		},
		Run: taskHistory,
	}, {
		Name: "requeue", Args: "-repo <dir> <id> [why]", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.requeue", "stop a run and put the task back in to do")
		},
		Run: requeueTask,
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

		// A constant this binary was built with, and what somebody types to
		// find out what they are running.
		OffRecord: true,
	}, {
		Name:  "upgrade",
		About: func(p *words.Printer) string { return p.T("cmd.upgrade", "check for updates and upgrade orbit") },
		Run:   upgrade,

		// The door the record's own refusal points at.
		OffRecord: true,
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
	}}, controlling(), answering()))
}

// lookup finds a command by the name that was typed.
func lookup(name string) (Command, bool) {
	for _, c := range commands() {
		if c.Name == name {
			return c, true
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
