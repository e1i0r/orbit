package cli

// orbit export: the record, written back out as the files it used to be.

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/export"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// exportRecord writes the whole record, or one task of it, into a directory
// as JSONL.
//
// The directory is a positional argument and not a flag because it is the
// whole of what this command is asked: `orbit export ~/orbit-backup` reads
// as the sentence somebody means, and there is nothing else to say. -task
// narrows it, which is the flag-shaped part.
func exportRecord(ctx Context, args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	only := fs.String("task", "", "write out one task rather than the whole record")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	p := ctx.printer()

	if fs.NArg() != 1 {
		return errors.New(p.T("export.needs_dir", "export needs one directory to write the record into"))
	}

	s, err := store.Open()
	if err != nil {
		return err
	}

	dir := fs.Arg(0)

	out, err := export.Run(s, dir, *only)
	if out.Tasks == 0 && out.Events == 0 && out.Messages == 0 {
		return err
	}

	// What came out is said even when something did not, because the two
	// answer different questions. The error names the tasks that stayed in
	// the file; this line is what somebody about to trust the directory
	// needs to read, and printing it only on a clean run would leave the
	// salvage of a damaged record looking like a failure that wrote nothing.
	fmt.Fprintf(ctx.Out, "%s\n", p.T("export.wrote", "wrote {what} into {dir}",
		words.Arg{Name: "what", Value: counted(p, out.Tasks, out.Events, out.Messages)},
		words.Arg{Name: "dir", Value: dir}))

	return err
}
