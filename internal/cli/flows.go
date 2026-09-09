package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
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
	// The language is the saved setting and nothing else. words.Resolve
	// falls through to $LANG, and a listing that changed language with the
	// terminal it was run in would make this command's own tests depend on
	// the machine running them. The flag and the environment variable join
	// in at the composition root, once, where every command can see them.
	cfg, err := s.Settings()
	if err != nil {
		return err
	}

	p := words.For(cfg.Language)
	for _, f := range flow.List(s) {
		fmt.Fprintf(ctx.Out, "%s (%s)\n", f.Name, verb.FlowMark(p, f.Origin))
	}

	return nil
}
