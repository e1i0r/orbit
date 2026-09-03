package cli

// orbit digest: what the record says about the work as a whole, in the
// figures worth acting on.

import (
	"flag"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
)

// digest reads every task in the state root and prints what they add up to.
//
// Every task and not a window of them. A digest of the last seven days needs
// a clock in the fold and an argument on the command, and what a reader
// actually asks first is whether any of this is working at all — which is a
// question about everything that has run.
func digest(ctx Context, args []string) error {
	fs := flag.NewFlagSet("digest", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	s, err := store.Open()
	if err != nil {
		logger.Error("cli/digest", "open the state root failed: %v", err)
		return err
	}

	ids, err := s.TaskIDs()
	if err != nil {
		logger.Error("cli/digest", "list the tasks failed: %v", err)
		return err
	}

	d := view.Digest{}

	for _, id := range ids {
		events, err := task.Events(s, task.Task{ID: id})
		if err != nil {
			// A record that will not read is one task missing from a
			// report, and a report that refuses to be printed because of it
			// is a report nobody gets.
			logger.Warn("cli/digest", "read the record of %q: %v", id, err)
			continue
		}

		d = view.Digested(d, events)
	}

	return printDigest(ctx, d, len(ids))
}

// printDigest lays the figures out in the order a reader asks for them: what
// landed, what it cost, what was thrown away, and where people keep having
// to step in.
func printDigest(ctx Context, d view.Digest, tasks int) error {
	p := ctx.printer()

	w := tabwriter.NewWriter(ctx.Out, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "%s\t%d of %d\n", p.T("digest.merged", "merged"), d.Merged, tasks)
	fmt.Fprintf(w, "%s\t$%.2f\n", p.T("digest.spent_merged", "what merging cost"), d.SpentMerged)
	fmt.Fprintf(w, "%s\t%d of %d\n", p.T("digest.untouched", "finished with nobody asked"), d.Untouched, d.Finished)
	fmt.Fprintf(w, "%s\t%d\t$%.2f\n", p.T("digest.stuck", "stuck"), d.Stuck, d.SpentStuck)
	fmt.Fprintf(w, "%s\t%d\n", p.T("digest.requeued", "sent back"), d.Requeued)
	fmt.Fprintf(w, "%s\t$%.2f\n", p.T("digest.spent", "spent in total"), d.Spent)

	if err := w.Flush(); err != nil {
		return err
	}

	ranked(ctx, p.T("digest.asked_at", "where people are stopped"), d.Asked())
	ranked(ctx, p.T("digest.rounds", "phases that go round again"), d.Rounds())

	return nil
}

// ranked prints one of the two lists, and nothing at all when it is empty.
//
// A heading over no rows reads as a fact about the work — that nobody is
// ever stopped — when it is a fact about a record with nothing in it yet.
func ranked(ctx Context, head string, counts []view.Count) {
	if len(counts) == 0 {
		return
	}

	fmt.Fprintf(ctx.Out, "\n%s\n", head)

	w := tabwriter.NewWriter(ctx.Out, 0, 0, 2, ' ', 0)
	for _, c := range counts {
		fmt.Fprintf(w, "  %s\t%d\n", c.Name, c.N)
	}

	_ = w.Flush() //nolint:errcheck // a writer that failed has already failed above
}
