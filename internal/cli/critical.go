package cli

// The critical protocol at the one boundary Orbit crosses on its own: the
// push that puts a task's work on somebody else's remote.

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// errNeedsPermission is what a delivery answers with when the task is
// critical and nobody has said yes yet.
//
// It is an error and not a quiet return: the command did not do what it was
// asked, and a reader who typed `orbit pr` and got a zero exit code would
// believe a pull request exists.
type errNeedsPermission struct{ said string }

func (e errNeedsPermission) Error() string { return e.said }

// permitted is the gate in front of one repository's push.
//
// Nothing inside the worktree asks for anything — editing, testing,
// installing are the sandbox, and the diff is where they are reviewed. This
// is the boundary: past here the work is on a remote other people pull from,
// and that is the only place a critical task stops.
//
// The order is the protocol's and cannot be rearranged. The state is written
// down, the backup is made, and only then is the question put — a reader
// asked to approve something with no backup behind it is being asked to
// approve something they cannot take back.
func permitted(ctx Context, s *store.Store, t task.Task, r repo.Repo, wtDir string) error {
	if !task.Critical(s, t) {
		return nil
	}

	a := task.Action{
		Name: "pr",
		Plan: fmt.Sprintf("push %s to %s and open a pull request", branchOf(t), r.Remote),
	}

	a, err := task.Snapshot(s, t, r, wtDir, a)
	if err != nil {
		return err
	}

	if task.Permitted(s, t, a) {
		return nil
	}

	p := ctx.printer()
	fmt.Fprint(ctx.Err, task.Refused(a))
	fmt.Fprintln(ctx.Err, p.T("critical.how", "allow it with `orbit permit {id}`, or refuse it with `orbit permit -no {id}`",
		words.Arg{Name: "id", Value: t.ID}))

	return errNeedsPermission{said: p.T("critical.refused", "{id} is critical and {action} has not been allowed",
		words.Arg{Name: "id", Value: t.ID},
		words.Arg{Name: "action", Value: a.Name})}
}

// noteCritical writes down that the action was taken and where it left
// things, for a task that went through the protocol.
//
// Best-effort and silent for an ordinary task: the summary is the fifth step
// of a protocol only critical tasks walk, and a pull request that was opened
// is opened whether or not the record could be written afterwards.
func noteCritical(s *store.Store, t task.Task, r repo.Repo, wtDir, url string) {
	if !task.Critical(s, t) {
		return
	}

	a, _ := task.Waiting(s, t)
	a.Name, a.Repo = "pr", r.Name

	if a.Revert == "" {
		a.Revert = fmt.Sprintf("gh pr close %s --delete-branch", branchOf(t))
	}

	after, err := r.HeadSHA(wtDir)
	if err != nil {
		logger.Warn("cli/pr", "read where %q stands after the push: %v", r.Name, err)
	}

	if url != "" {
		a.Plan = url
	}

	if err := task.Applied(s, t, a, after); err != nil {
		logger.Warn("cli/pr", "write down what %q did to %q: %v", t.ID, r.Name, err)
	}
}

// permitTask is the reader answering: yes, or no.
//
// One command with a flag rather than two verbs, because it is one decision
// with two answers and a reader looking for how to refuse should find it
// beside how to allow.
func permitTask(ctx Context, args []string) error {
	fs := flag.NewFlagSet("permit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	no := fs.Bool("no", false, "refuse the action instead of allowing it")

	by := fs.String("by", "operator", "who is answering")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	id := fs.Arg(0)
	if id == "" {
		return needsTaskID(ctx, "permit")
	}

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		logger.Error("cli/permit", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		logger.Error("cli/permit", "load task %q failed: %v", id, err)
		return err
	}

	p := ctx.printer()

	a, waiting := task.Waiting(s, t)
	if !waiting {
		fmt.Fprintln(ctx.Out, p.T("permit.nothing", "{id} is not waiting on you for anything",
			words.Arg{Name: "id", Value: t.ID}))

		return nil
	}

	if err := task.Answer(s, t, a, !*no, *by); err != nil {
		return err
	}

	if *no {
		fmt.Fprintln(ctx.Out, p.T("permit.refused", "{action} on {id} was refused; nothing left this machine",
			words.Arg{Name: "action", Value: a.Name},
			words.Arg{Name: "id", Value: t.ID}))

		return nil
	}

	fmt.Fprintln(ctx.Out, p.T("permit.allowed", "{action} on {id} is allowed; run it again to do it",
		words.Arg{Name: "action", Value: a.Name},
		words.Arg{Name: "id", Value: t.ID}))

	return nil
}

// criticalTask marks a task as one that reaches something that matters, or
// takes the mark off again.
func criticalTask(ctx Context, args []string) error {
	fs := flag.NewFlagSet("critical", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	off := fs.Bool("off", false, "take the mark off again")

	by := fs.String("by", "operator", "who is marking it")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	id := fs.Arg(0)
	if id == "" {
		return needsTaskID(ctx, "critical")
	}

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		logger.Error("cli/critical", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		logger.Error("cli/critical", "load task %q failed: %v", id, err)
		return err
	}

	if err := task.Mark(s, t, !*off, *by); err != nil {
		return err
	}

	p := ctx.printer()
	if *off {
		fmt.Fprintln(ctx.Out, p.T("critical.off", "{id} is an ordinary task again",
			words.Arg{Name: "id", Value: t.ID}))

		return nil
	}

	fmt.Fprintln(ctx.Out, p.T("critical.on",
		"{id} is critical: before it pushes anything, Orbit writes down where things stand, tags a backup and asks you",
		words.Arg{Name: "id", Value: t.ID}))

	return nil
}
