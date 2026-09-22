package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/verb"
)

// flows lists the names a task can be written against.
//
// It exists because a name is the whole of the interface to a flow: `orbit
// new -flow careful` and `orbit set flow careful` both take a word, and a
// word you cannot look up is a word you get wrong. Each line says where the
// flow that name resolves to came from — the binary, or a file of the
// user's — so a flow that stopped behaving as documented is one command
// away from explaining itself.
//
// There is no -repo flag: flows are the user's and not a repository's, and
// they live at the root of the state tree beside the settings.
func flows(ctx Context, args []string) error {
	fs := flag.NewFlagSet("flows", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	s, err := store.Open()
	if err != nil {
		return err
	}

	// The Context's printer and not a second reading of the settings file:
	// this command used to open that file for the language itself, and so
	// was the one listing $ORBIT_LANG could not reach even after every other
	// command could. Whatever the dispatcher weighed, this speaks.
	p := ctx.printer()
	for _, f := range flow.List(s) {
		fmt.Fprintf(ctx.Out, "%s (%s)\n", f.Name, verb.FlowMark(p, f.Origin))
	}

	return nil
}
