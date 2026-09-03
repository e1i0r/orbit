package task

// The critical protocol: what has to be true before a task is allowed to do
// something that cannot be undone.

import (
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// Critical says whether this task has been marked as touching something that
// matters — the tasks that have to be done and have to be done carefully.
//
// It is a mark on the task and not on a flow or a phase. What makes a piece
// of work dangerous is the data it reaches, which the person writing the
// task knows and no file of phases does: the same flow is careless against a
// scratch database and careful against the ledger.
//
// A task nobody marked goes through none of this. That is the whole reason
// the mark exists — a protocol that applied to everything would be a
// permission prompt on every push, and the tenth one is approved without
// being read.
func Critical(s *store.Store, t Task) bool {
	events, err := Events(s, t)
	if err != nil {
		return false
	}

	on := false

	for _, e := range events {
		if e.Kind == record.TaskCritical {
			on = e.Data["on"] != "false"
		}
	}

	return on
}

// Mark turns the protocol on or off for a task, and writes down that
// somebody did.
//
// It can be turned off, and that is not a hole: the reader who marked a task
// by mistake would otherwise have no way back, and every event of it stays
// in the record — what changes is what happens next, never what happened.
func Mark(s *store.Store, t Task, on bool, by string) error {
	return emit(s, t, record.Event{
		Kind: record.TaskCritical,
		Data: map[string]string{"on": fmt.Sprint(on), "by": by},
	})
}

// Action is one thing a critical task is about to do that cannot be undone,
// and everything the protocol needs to say about it.
type Action struct {
	// Name is the verb, as the reader typed it: pr, merge.
	Name string
	// Repo is where it happens, and Ref the commit the world is at before
	// it does.
	Repo string
	Ref  string
	// Plan is what it will do, in one line, and Revert is the command that
	// undoes it.
	Plan   string
	Revert string
}

// Snapshot writes down how the world stands before a critical action, and
// backs up what the action is about to move.
//
// The backup is not a step the caller can skip. A gate that asked for
// permission without one would be asking a reader to approve something they
// could not take back, which is the opposite of what this protocol is for —
// so the backup is made first and its ref is what the gate shows.
//
// The ref is a git tag under orbit/backup/, which is a name that survives a
// branch being deleted, a force-push, and the worktree being thrown away.
func Snapshot(s *store.Store, t Task, r repo.Repo, wtDir string, a Action) (Action, error) {
	head, err := r.HeadSHA(wtDir)
	if err != nil {
		return a, fmt.Errorf("read where %q stands before %s: %w", r.Name, a.Name, err)
	}

	a.Repo, a.Ref = r.Name, head
	if a.Revert == "" {
		a.Revert = fmt.Sprintf("git -C %s reset --hard %s", wtDir, head)
	}

	if err := emit(s, t, record.Event{
		Kind: record.CriticalSnapshot,
		Text: a.Plan,
		Data: map[string]string{"action": a.Name, "repo": r.Name, "ref": head},
	}); err != nil {
		return a, err
	}

	name := backupRef(t.ID)
	if err := r.Backup(wtDir, name, head); err != nil {
		return a, fmt.Errorf("back %q up before %s: %w", r.Name, a.Name, err)
	}

	return a, emit(s, t, record.Event{
		Kind: record.CriticalBackup,
		Data: map[string]string{"action": a.Name, "repo": r.Name, "ref": head, "backup": name},
	})
}

// backupRef is where a task's backup lives, named so that two of them on one
// day do not overwrite each other and so that a reader can see whose it is.
func backupRef(id string) string {
	return fmt.Sprintf("orbit/backup/%s/%s", id, time.Now().UTC().Format("20060102-150405"))
}

// Permitted says whether a reader has approved this exact action at this
// exact commit.
//
// At this commit, because approval is of a plan and a plan is about a state
// of the world. Work that moved after the yes was given is work nobody said
// yes to, and asking again is cheaper than pushing something a reader never
// saw.
func Permitted(s *store.Store, t Task, a Action) bool {
	events, err := Events(s, t)
	if err != nil {
		return false
	}

	for _, e := range events {
		if e.Kind != record.CriticalApproved {
			continue
		}

		if e.Data["action"] == a.Name && e.Data["ref"] == a.Ref && e.Data["repo"] == a.Repo {
			return true
		}
	}

	return false
}

// Waiting is the action a task is waiting to be allowed to do, and nothing
// when it is waiting on nobody.
//
// The newest snapshot that has no approval and no rejection after it. A task
// asked twice — a push that was refused and then asked again — is waiting on
// the second question, and answering the first would approve a plan the
// reader has already seen the end of.
func Waiting(s *store.Store, t Task) (Action, bool) {
	events, err := Events(s, t)
	if err != nil {
		return Action{}, false
	}

	var (
		open   Action
		asked  bool
		backed bool
	)

	for _, e := range events {
		switch e.Kind {
		case record.CriticalSnapshot:
			open = Action{Name: e.Data["action"], Repo: e.Data["repo"], Ref: e.Data["ref"], Plan: e.Text}
			asked, backed = true, false
		case record.CriticalBackup:
			backed = true
		case record.CriticalApproved, record.CriticalRejected, record.CriticalApplied:
			asked, backed = false, false
		}
	}

	// A snapshot with no backup after it is a question that was never
	// finished being asked: the process died between the two, and what the
	// reader would be approving has nothing behind it.
	return open, asked && backed
}

// Answer records a reader's yes or no to the action a task is waiting on.
func Answer(s *store.Store, t Task, a Action, yes bool, by string) error {
	kind := record.CriticalRejected
	if yes {
		kind = record.CriticalApproved
	}

	return emit(s, t, record.Event{
		Kind: kind,
		Text: a.Plan,
		Data: map[string]string{"action": a.Name, "repo": a.Repo, "ref": a.Ref, "by": by, "revert": a.Revert},
	})
}

// Applied writes down that the action was taken, and how the world stands
// now.
//
// Before and after both, because the summary a reader needs afterwards is
// the difference between them — and because an action that reported success
// while leaving the world where it was is the failure this protocol exists
// to catch.
func Applied(s *store.Store, t Task, a Action, after string) error {
	return emit(s, t, record.Event{
		Kind: record.CriticalApplied,
		Text: a.Plan,
		Data: map[string]string{
			"action": a.Name, "repo": a.Repo,
			"ref": a.Ref, "now": after, "revert": a.Revert,
		},
	})
}

// Refused is the sentence a command says when it will not go on, and it says
// how to allow it.
func Refused(a Action) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s in %s needs your word first: %s\n", a.Name, a.Repo, a.Plan)
	fmt.Fprintf(&b, "before: %s\n", a.Ref)
	fmt.Fprintf(&b, "undo:   %s\n", a.Revert)

	return b.String()
}
