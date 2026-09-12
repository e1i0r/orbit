package cli

import (
	"flag"
	"fmt"
	"io"
)

// Version is the release orbit was built at. It is a var, not a const, so a
// build can set it with -ldflags "-X ...cli.Version=v1.2.3"; the default,
// "dev", is what `go build` and `go run` leave it at, which is itself the
// honest answer for a checkout with no release tag.
var Version = "dev"

// version prints the release orbit was built at, under the mark from the
// logo: a body with rings around it. It takes no flags of its own; parse
// still runs so `-h` shows the same shape every other command does, and an
// unknown flag is refused the same way.
func version(ctx Context, args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	// The mark, the name with the build, and the tagline in the reader's
	// language — the key already exists for the usage screen, so no new
	// entry. Colour only on a terminal of its own; see banner.go.
	p := ctx.printer()
	fmt.Fprint(ctx.Out, banner(Version,
		p.T("cli.tagline", "orbit — a cockpit for supervising coding agents"),
		useColor(ctx.Out)))

	return nil
}
