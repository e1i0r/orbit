package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/e1i0r/orbit/internal/repo"
)

func repos(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("repos", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := parse(fs, args, out); err != nil {
		return err
	}
	root := fs.Arg(0)
	if root == "" {
		var err error
		if root, err = os.Getwd(); err != nil {
			return fmt.Errorf("locate the working directory: %w", err)
		}
	}
	found, err := repo.Discover(root)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		fmt.Fprintf(out, "no repositories under %s\n", root)
		return nil
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, r := range found {
		remote := r.Remote
		if remote == "" {
			remote = "—"
		}
		base := r.Base
		if base == "" {
			// An empty Base is repo.Open saying the checkout is not on a
			// branch. Saying so is the point; a blank column is not.
			base = "detached"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Name, remote, base, r.Path)
	}
	return w.Flush()
}
