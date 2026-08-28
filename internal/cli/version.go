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

// version prints the release orbit was built at. It takes no flags of its
// own; parse still runs so `-h` shows the same shape every other command
// does, and an unknown flag is refused the same way.
func version(ctx Context, args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	fmt.Fprintf(ctx.Out, "orbit %s\n", Version)

	return nil
}
