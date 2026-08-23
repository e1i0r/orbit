// Package cli is the command line: one file per command, and a dispatcher
// that owns no logic of its own.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// synopsis is the whole interface, in the order it is worth reading. It is a
// table rather than a block of text because the columns have to line up on
// their own: the `new` line is longer than the rest, and the hand-counted
// spaces that once aligned them stopped aligning the moment it was written.
var synopsis = [][2]string{
	{"orbit repos [dir]", "list the repositories under a directory"},
	{"orbit new -repo <dir> -id <id> <text>", "write a task down"},
	{"orbit run -repo <dir> <id>", "run a task through its flow"},
	{"orbit pause -repo <dir> <id>", "stop a run at its next phase"},
	{"orbit resume -repo <dir> <id>", "let a stopped run carry on"},
	{"orbit list -repo <dir>", "list the tasks of a repository"},
	{"orbit show -repo <dir> <id>", "print what happened to a task"},
	{"orbit read -repo <dir> <id>", "mark a finished task as looked at"},
	{"orbit cancel -repo <dir> <id>", "stop a run, and say so in its record"},
	{"orbit reconcile -repo <dir> [id]", "close the records of runs whose processes are gone"},
	{"orbit set <key> <value>", "change a setting"},
}

// usage is the whole of orbit on one screen.
func usage() string {
	var b strings.Builder
	b.WriteString("orbit — a cockpit for supervising coding agents\n\n")
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	for _, s := range synopsis {
		fmt.Fprintf(w, "  %s\t%s\n", s[0], s[1])
	}
	_ = w.Flush() // a strings.Builder cannot fail to be written to
	b.WriteString("\nState lives in $ORBIT_HOME, or ~/.orbit when that is unset.\n")
	return b.String()
}

// errHelpShown says a subcommand's only job this time was to print its own
// flags. Asking a program what it takes is not a failure, so Run turns this
// into exit code 0 and prints nothing further.
var errHelpShown = errors.New("help shown")

// parse reads a subcommand's flags and makes flag's two failure modes
// legible.
//
// Every set is built with ContinueOnError and its output discarded, so flag
// prints nothing itself: `orbit new -h` came back as the error "flag: help
// requested", which the dispatcher printed as `orbit: flag: help requested`
// and exited 1 with the flags never shown, and `orbit list -repos .` said
// what was wrong and then offered nothing. For a tool whose entire interface
// is flags, that is the first thing a stranger meets.
func parse(fs *flag.FlagSet, args []string, out io.Writer) error {
	err := fs.Parse(args)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, flag.ErrHelp):
		help(fs, out)
		return errHelpShown
	default:
		var b strings.Builder
		help(fs, &b)
		return fmt.Errorf("%w\n\n%s", err, strings.TrimRight(b.String(), "\n"))
	}
}

// help writes one subcommand's shape and its flags.
func help(fs *flag.FlagSet, out io.Writer) {
	for _, s := range synopsis {
		if s[0] == "orbit "+fs.Name() || strings.HasPrefix(s[0], "orbit "+fs.Name()+" ") {
			fmt.Fprintf(out, "%s — %s\n\n", s[0], s[1])
		}
	}
	fs.SetOutput(out)
	fs.PrintDefaults()
	fs.SetOutput(io.Discard)
}

// Run dispatches one command and returns the exit code.
//
// The writers are parameters and the code is returned rather than passed to
// os.Exit so that every command can be tested in-process, without building a
// binary and without a subprocess.
func Run(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(out, usage())
		return 2
	}
	var err error
	switch args[0] {
	case "repos":
		err = repos(args[1:], out)
	case "new":
		err = newTask(args[1:], out)
	case "run":
		err = runTask(args[1:], out)
	case "pause", "resume":
		// The word the reader typed is the word the run is left, which is
		// why these two share a case: the command name is the argument.
		err = controlTask(args[0], args[1:], out)
	case "list":
		err = list(args[1:], out)
	case "show":
		err = show(args[1:], out)
	case "read":
		err = readTask(args[1:], out)
	case "cancel":
		err = cancelTask(args[1:], out)
	case "reconcile":
		err = reconcile(args[1:], out)
	case "set":
		err = set(args[1:], out)
	case "help", "-h", "--help":
		fmt.Fprint(out, usage())
		return 0
	default:
		fmt.Fprintf(errOut, "orbit: %q is not a command\n\n%s", args[0], usage())
		return 2
	}
	if errors.Is(err, errHelpShown) {
		return 0
	}
	if err != nil {
		fmt.Fprintf(errOut, "orbit: %v\n", err)
		return 1
	}
	return 0
}

// openBoth resolves the repository and the store together, since every
// command that touches a task needs exactly these two. The repository is
// opened first: it is the part supplied by the user and most likely to be
// wrong, and store.Open creates $ORBIT_HOME as a side effect of merely
// opening it, which a bad -repo must not leave behind.
func openBoth(dir string) (*store.Store, repo.Repo, error) {
	r, err := repo.Open(dir)
	if err != nil {
		return nil, repo.Repo{}, err
	}
	s, err := store.Open()
	if err != nil {
		return nil, repo.Repo{}, err
	}
	return s, r, nil
}
