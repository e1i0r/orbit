package cli

// top's flag grammar, on its own, so that something can ask it what it read.
//
// It is a file of its own for one reason. While the three flag declarations
// lived inside top(), the only way to find out what they had been set to was
// to run the command — and every test of `orbit top` reaches the plain frame
// through interactive(out), which refuses any writer that is not the
// process's own stdout. A test hands the command a buffer, so it takes the
// plain branch whether or not -once was ever read, and -once is never the
// deciding term. The headline flag of this command had no test that could
// fail: neutralising it entirely left the whole suite green.
//
// A function that answers with what it parsed is testable without a
// terminal, and that is the whole of the fix. top() reads the answer; this
// file decides it.

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/words"
)

// parseTop reads top's flags with the directory allowed on either side of
// them, and answers with everything it read.
//
// flag stops at the first argument that is not a flag, so `orbit top ~/work
// -once` would leave -once unread — and the failure would not be a message
// but a full-screen window where a frame was asked for, in a pipe, in CI.
// Every other command in this program takes its directory as -repo and never
// meets this; top's directory is the thing a person types most often, and
// making them type a flag for it to be read is the wrong half of the trade.
//
// So: parse, take one positional, parse what is left, until nothing is left.
// A second directory is refused rather than one of them being chosen
// silently — two roots is a person meaning something this command cannot do,
// and picking the first is how they find out an hour later.
//
// ctx is where `orbit top -h` prints the flags, and nothing else about the
// command is read out of it: the flag set discards its own output so that
// flag's two failure modes come back as errors this program words itself.
// See parse.
func parseTop(ctx Context, args []string) (dir string, once bool, lang string, err error) {
	fs := flag.NewFlagSet("top", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	onceFlag := fs.Bool("once", false, "draw one frame as plain text and exit — what a pipe, a log or CI reads")
	langFlag := fs.String("lang", "", "draw in this language, e.g. es; otherwise $ORBIT_LANG, then the saved setting, then $LANG")

	var dirs []string

	rest := args
	for {
		if perr := parse(ctx, fs, rest); perr != nil {
			return "", false, "", perr
		}

		rest = fs.Args()
		if len(rest) == 0 {
			break
		}

		dirs = append(dirs, rest[0])
		rest = rest[1:]
	}

	switch len(dirs) {
	case 0:
		cwd, werr := os.Getwd()
		if werr != nil {
			return "", false, "", fmt.Errorf("%s: %w", ctx.printer().T("cli.locate_working_directory",
				"locate the working directory"), werr)
		}

		dir = cwd
	case 1:
		dir = dirs[0]
	default:
		return "", false, "", errors.New(ctx.printer().T("top.one_directory",
			"top takes one directory, and was given {count}: {dirs}",
			words.Arg{Name: "count", Value: strconv.Itoa(len(dirs))},
			words.Arg{Name: "dirs", Value: strings.Join(dirs, " ")}))
	}

	return dir, *onceFlag, *langFlag, nil
}
