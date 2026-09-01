// Package cli is the command line: one file per command, one table in
// commands.go saying which commands there are, and a dispatcher that owns no
// logic of its own.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// errHelpShown says a subcommand's only job this time was to print its own
// flags. Asking a program what it takes is not a failure, so Run turns this
// into exit code 0 and prints nothing further.
var errHelpShown = errors.New("help shown")

// needsTaskID is the refusal three commands share. It is one sentence with
// the command's own name in it rather than three sentences differing by a
// word: a command name is typed rather than read, so it is the one thing in
// a translated sentence that must not be translated.
func needsTaskID(ctx Context, command string) error {
	return errors.New(ctx.printer().T("cli.needs_task_id", "{command} needs the id of a task",
		words.Arg{Name: "command", Value: command}))
}

// parse reads a subcommand's flags and makes flag's two failure modes
// legible.
//
// Every set is built with ContinueOnError and its output discarded, so flag
// prints nothing itself: `orbit new -h` came back as the error "flag: help
// requested", which the dispatcher printed as `orbit: flag: help requested`
// and exited 1 with the flags never shown, and `orbit list -repos .` said
// what was wrong and then offered nothing. For a tool whose entire interface
// is flags, that is the first thing a stranger meets.
func parse(ctx Context, fs *flag.FlagSet, args []string) error {
	err := fs.Parse(args)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, flag.ErrHelp):
		help(ctx, fs)
		return errHelpShown
	default:
		var b strings.Builder
		help(Context{Out: &b, Words: ctx.Words}, fs)

		return fmt.Errorf("%w\n\n%s", err, strings.TrimRight(b.String(), "\n"))
	}
}

// help writes one subcommand's shape and its flags.
//
// The shape comes out of the table by name rather than by scanning the usage
// screen for a line that starts with the right words, which is what this did
// before: a command whose name was a prefix of another's printed both.
func help(ctx Context, fs *flag.FlagSet) {
	if c, ok := lookup(fs.Name()); ok {
		fmt.Fprintf(ctx.Out, "%s — %s\n\n", c.Usage(), c.About(ctx.printer()))
	}

	fs.SetOutput(ctx.Out)
	fs.PrintDefaults()
	fs.SetOutput(io.Discard)
}

// Run dispatches one command and returns the exit code.
//
// The writers are parameters and the code is returned rather than passed to
// os.Exit so that every command can be tested in-process, without building a
// binary and without a subprocess.
func Run(args []string, out, errOut io.Writer) (code int) {
	ctx := Context{Out: out, Err: errOut, Words: printer()}
	if len(args) == 0 {
		return Run([]string{"top"}, out, errOut)
	}

	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(out, usage(ctx.Words))
		return 0
	}

	c, ok := lookup(args[0])
	if !ok {
		fmt.Fprintf(errOut, "orbit: %q is not a command\n\n%s", args[0], usage(ctx.Words))
		return 2
	}

	defer logging(errOut, &code)()

	// The name and not the arguments. A description typed at `orbit new` is
	// prose somebody wrote, and prose in a diagnostic log is a paragraph
	// between two facts; what the command was asked to act on is in the
	// error below when it matters, and in the record always.
	logger.Info("cli/"+c.Name, "ran")

	flatten(errOut)
	migrateRecord(errOut)

	err := c.Run(ctx, args[1:])
	if errors.Is(err, errHelpShown) {
		return 0
	}

	if err != nil {
		// Every command's failure, in one place, for the same reason the log
		// is opened in one place: a command remembering to write its own down
		// is a chance to forget, and nine of the twenty-two took it — flows,
		// list, read, repos, resume, settings, show, top and upgrade each
		// failed silently as far as any file was concerned. The other
		// thirteen now say it twice, in their own words and then in these,
		// and a line repeated is a far smaller thing than a line missing.
		logger.Error("cli/"+c.Name, "%v", err)
		fmt.Fprintf(errOut, "orbit: %v\n", err)

		return 1
	}

	return 0
}

// logging opens the log for the length of one command, and returns the
// function that closes it.
//
// It is here, in the dispatcher, because the version this replaces had it
// in two commands out of nineteen. `orbit run` — the command that actually
// puts an agent on a repository — wrote every one of its failures to a nil
// global logger, which drops them without a word; so did cancel, merge,
// pr, close-pr and reconcile. Sixty-one logger calls existed and went
// nowhere unless you happened to be inside the window or the mcp server.
//
// Nothing is said about the log until the command is over. The window
// spends its whole life on the alternate screen, and a sentence printed
// before it opens is a sentence wiped a millisecond later; printed here,
// after it hands the terminal back, it is a sentence somebody reads.
func logging(errOut io.Writer, code *int) func() {
	trouble := func() error {
		s, err := store.Open()
		if err != nil {
			return err
		}

		return logger.Init(s.LogPath(), s.ErrorLogPath())
	}()

	return func() {
		err := errors.Join(trouble, logger.CloseGlobal())
		if err == nil {
			return
		}

		fmt.Fprintln(errOut, "orbit: the log:", err)

		// A command that did its work and wrote nothing down did not
		// entirely succeed. This is how a reader finds out today instead of
		// next week, from an empty file, while looking for why something
		// else broke — and a code the command set itself is left alone,
		// because what it was asked to do matters more than the log.
		if *code == 0 {
			*code = 1
		}
	}
}

// printer is the language everything orbit prints is in.
//
// It is the saved setting and nothing else. words.Resolve falls through to
// $LANG, and a command whose output changed language with the terminal it
// was run in would make this package's own tests depend on the machine
// running them. The flag and the environment variable join in at the
// window's composition root, once, where there is a person watching — see
// top.go.
func printer() *words.Printer { return words.For(language()) }

// language reads the saved language without creating anything.
//
// The state root is looked for before it is opened, and that is the whole
// point of this function: store.Open makes $ORBIT_HOME as a side effect of
// merely opening it, and `orbit help` — which a stranger types before they
// have run anything — must not leave a folder behind for having been asked a
// question. Every failure answers with the empty string, which words.For
// reads as the English written at each call site.
func language() string {
	root, err := store.RootPath()
	if err != nil {
		return ""
	}

	if _, err := os.Stat(root); err != nil {
		return ""
	}

	s, err := store.Open()
	if err != nil {
		return ""
	}

	cfg, err := s.Settings()
	if err != nil {
		return ""
	}

	return cfg.Language
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
