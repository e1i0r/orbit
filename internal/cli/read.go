package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
)

// readTask writes down that somebody has looked at a finished task.
//
// It is the other half of the unread cap: the cap counts finished work
// nobody has read, and this is the only thing that lowers that count. A
// brake with no release is a brake people disable, so the release is a
// command and not a gesture only the window has.
func readTask(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("read", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(fs, args, out); err != nil {
		return err
	}
	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("read needs the id of a task")
	}

	s, r, err := openBoth(*dir)
	if err != nil {
		return err
	}
	t, err := task.Load(s, r, id)
	if err != nil {
		return err
	}
	if err := task.MarkRead(s, t); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s marked read\n", id)
	return nil
}

// Unread counts the finished work nobody has looked at, which is the number
// task.Start refuses at.
//
// It lives here rather than in internal/task because it is a question about
// the whole board and internal/board already imports internal/task: a count
// on the other side would be an import cycle, and — worse — a second fold of
// the record beside internal/view's. This one walks a Board that has already
// been folded, so the number and the rows on screen cannot disagree.
//
// Cancelled is not unread. Its band is Done because the reader is the one
// who stopped it, and asking somebody to acknowledge the thing they just
// cancelled is how a brake earns its reputation for being in the way. A
// failed run is not counted either, and for a different reason: it is not in
// Done at all — internal/view bands it as NeedsYou, where it is already in
// front of the reader without a counter's help.
func Unread(b board.Board) int {
	n := 0
	for _, t := range b.Tasks {
		if view.BandOf(t) != view.Done {
			continue
		}
		if t.Read || t.Reason.Key == view.ReasonCancelled {
			continue
		}
		n++
	}
	return n
}
