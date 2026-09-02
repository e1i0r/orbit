package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// controlTask leaves one word for a run to find at its next phase boundary.
//
// `orbit pause` and `orbit resume` are one function because they differ in
// exactly one string, and that string is also the name of the flag set — so
// a mistyped flag under `orbit resume` prints resume's own line out of the
// synopsis rather than pause's. Two copies of this body would be two places
// for the -repo flag's help text to drift apart.
//
// It returns as soon as the word is written, and says so. A run stopped at a
// gate takes it up within a poll; a run in the middle of a phase takes it up
// when that phase ends, which can be a long time — and a message that said
// "paused" would be claiming something that has not happened yet. A task
// nobody is running keeps the word for the run that starts next, which is
// the property that makes a file better than a signal here.
func controlTask(word string, ctx Context, args []string) error {
	fs := flag.NewFlagSet(word, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	id := fs.Arg(0)
	if id == "" {
		return needsTaskID(ctx, word)
	}

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		logger.Error("cli/pause", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		logger.Error("cli/pause", "load task %q in %q failed: %v", id, r.Name, err)
		return err
	}

	if err := task.Control(s, t, word); err != nil {
		logger.Error("cli/pause", "control task %q in %q (%s) failed: %v", id, r.Name, word, err)
		return err
	}

	logger.Info("cli/pause", "task %s in %s requested to %s", id, r.Name, word)
	fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("pause.asked",
		"{id} asked to {word} — a run in flight notices at its next phase",
		words.Arg{Name: "id", Value: id}, words.Arg{Name: "word", Value: word}))

	return nil
}
