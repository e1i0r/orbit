package cli

// What `orbit help` says, and what the log files a run under.
//
// Help is answered from the declaration by path: a family is asked for as
// both its words, and the log names both words for the same reason —
// `orbit task cancel` failing is not filed under every other thing
// `orbit task` does.

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/verb"
)

// help writes one subcommand's shape and its flags.
//
// The shape comes out of the declaration by path rather than by scanning
// the usage screen for a line that starts with the right words, which is
// what this did before: a command whose name was a prefix of another's
// printed both. A family is asked for as both its words — `orbit pr merge`
// — and answered with the child's own line.
func help(ctx Context, fs *flag.FlagSet) {
	if line, ok := usageFor(ctx, fs.Name()); ok {
		fmt.Fprintf(ctx.Out, "%s\n\n", line)
	} else if c, ok := lookup(fs.Name()); ok {
		fmt.Fprintf(ctx.Out, "%s — %s\n\n", c.Usage(), c.About(ctx.printer()))
	}

	fs.SetOutput(ctx.Out)
	fs.PrintDefaults()
	fs.SetOutput(io.Discard)
}

// usageFor is one verb's own line: what it is called with both its words,
// what it takes, and what it does. A parent answers with its children
// beside it, the way the usage screen lists them.
func usageFor(ctx Context, path string) (string, bool) {
	v, known := verb.One(path)
	if !known {
		return "", false
	}

	p := ctx.printer()
	line := "orbit " + v.Path() + argsOf(v) + " — " + v.About(p)

	if kids := v.Children(); len(kids) > 0 {
		names := make([]string, 0, len(kids))
		for _, kid := range kids {
			names = append(names, kid.Name)
		}

		line += "  |  " + strings.Join(names, "|")
	}

	return line, true
}

// commandPath is the command as the log names it: both words when a
// family's child was asked for, so that `orbit task cancel` failing is not
// filed under every other thing `orbit task` does.
func commandPath(c Command, args []string) string {
	if len(args) > 1 {
		if v, known := verb.One(c.Name); known {
			for _, kid := range v.Children() {
				if args[1] == kid.Name {
					return c.Name + " " + kid.Name
				}
			}
		}
	}

	return c.Name
}
