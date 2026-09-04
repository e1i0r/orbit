package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// controlling is the three commands that leave a word behind: pause, resume
// and skip.
//
// They are a slice of their own, appended to the table in commands.go, for
// the reason answering()'s two are: that file is at the size ceiling. Here
// they also sit beside the one function all three call.
//
// skip is the word the gate has always understood and nothing could write.
// It differs from resume in what happens to the phase the run is stopped in
// front of: resume runs it, skip does not, and the run carries on at the one
// after it.
func controlling() []Command {
	return []Command{{
		Name: "pause", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string { return p.T("cmd.pause", "stop a run at its next phase") },
		Run:   func(ctx Context, args []string) error { return controlTask("pause", ctx, args) },
	}, {
		Name: "resume", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string { return p.T("cmd.resume", "let a stopped run carry on") },
		Run:   func(ctx Context, args []string) error { return controlTask("resume", ctx, args) },
	}, {
		Name: "skip", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.skip", "let a stopped run carry on without the phase it is waiting in front of")
		},
		Run: func(ctx Context, args []string) error { return controlTask("skip", ctx, args) },
	}}
}

// controlTask leaves one word for a run to find at its next phase boundary.
//
// The three are one function because they differ in exactly one string, and
// that string is also the name of the flag set — so a mistyped flag under
// `orbit resume` prints resume's own line out of the synopsis rather than
// pause's. Three copies of this body would be three places for the -repo
// flag's help text to drift apart.
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
