package mcp

// The tools that write a task down and start it again, as adapters over
// the declaration.
//
// Writing and spending are two decisions even here: createTask never
// starts what it wrote, and retryTask reads the liveness first, because a
// model cannot look at the run file to settle whether a phase is running.
// What either means is the verb's, asked here the way every other door
// asks it.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/view"
)

// createTask writes a task down. It does not start it: writing a task and
// spending money on it are two decisions, and the board is where the second
// one is taken — by a reader, or by orbit_retry_task once somebody has read
// what was written.
func (sn Session) createTask(args map[string]any) CallToolResult {
	title := strings.TrimSpace(stringArg(args, "title"))
	if title == "" {
		return refuse(fmt.Errorf("this tool needs title"))
	}

	sb, err := sn.readBoard()
	if err != nil {
		return refuse(err)
	}

	defer sb.close()

	r, err := sn.pickRepo(sb.board, stringArg(args, "repo"))
	if err != nil {
		return refuse(err)
	}

	id, err := taskID(sb.store, r, stringArg(args, "id"))
	if err != nil {
		return refuse(err)
	}

	// Title and prompt are joined into the one thing a task is: the verbatim
	// text of task.md, which is everything the engines are told. They are
	// two arguments because a model writing a task naturally has a summary
	// and a body, and one field would get the summary alone.
	text := title
	if prompt := strings.TrimSpace(stringArg(args, "prompt")); prompt != "" {
		text = title + "\n\n" + prompt
	}

	out, err := verb.Run(sn.context(), sn.world(sb), "new", verb.In{
		Args: map[string]string{
			"id":   id,
			"text": text,
			"repo": r.Path,
			"flow": stringArg(args, "flow"),
		},
		By: journalBy,
	})
	if err != nil {
		return refuse(fmt.Errorf("write task %s down in %s: %w", id, r.Name, err))
	}

	_ = out

	t, err := task.Load(sb.store, r, id)
	if err != nil {
		return refuse(fmt.Errorf("load task %s: %w", id, err))
	}

	flow := stringArg(args, "flow")
	if flow == "" {
		flow = t.Flow
	}

	trace := journal(sb.store, t, "a model wrote this task down over mcp; nobody has started it")

	return reply(map[string]any{
		"id":        t.ID,
		"repo":      r.Name,
		"repo_path": r.Path,
		"flow":      flow,
		"band":      bandSlug(view.ToDo),
		"started":   false,
		"message":   fmt.Sprintf("task %s is written against %s and will walk the %s flow; start it with orbit_retry_task%s", t.ID, r.Name, flow, trace),
	})
}

// taskID is the id a new task gets: the one the caller chose, checked, or
// one minted from the repository's name.
func taskID(s *store.Store, r repo.Repo, chosen string) (string, error) {
	if chosen == "" {
		return nextTaskID(s, r)
	}

	if err := store.ValidTaskID(chosen); err != nil {
		return "", fmt.Errorf("id %q cannot be used: %w", chosen, err)
	}

	return chosen, nil
}

// retryTask runs a task that is not running.
//
// The liveness is read before asking, rather than letting the start refuse:
// a run that failed an hour ago has no process to let past, and a model
// cannot look at the run file to settle it, so it is told what to say to
// whoever can. The start itself refuses too — two readers of one marker
// would be how a second engine ends up in a worktree the first one is
// still writing in.
func (sn Session) retryTask(args map[string]any) CallToolResult {
	sb, err := sn.readBoard()
	if err != nil {
		return refuse(err)
	}

	defer sb.close()

	row0, err := findTask(sb.board, stringArg(args, "task_id"))
	if err != nil {
		return refuse(err)
	}

	if row0.Live == view.LiveHeld {
		return refuse(fmt.Errorf("task %s is running; pause it with orbit_pause_task or stop it with orbit_cancel_task before starting it again", row0.ID))
	}

	// Neither running nor free: the marker is there and nothing could read
	// it. Starting anyway is how a second engine ends up in a worktree the
	// first one is still writing in, and a model cannot look at the file to
	// settle it, so it is told what to say to whoever can.
	if row0.Live == view.LiveUnknown {
		return refuse(fmt.Errorf("orbit cannot read task %s's run marker, so it cannot tell whether a phase is running; the run file in the task's directory has to be looked at before it is started again", row0.ID))
	}

	r, err := openTaskRepo(row0)
	if err != nil {
		return refuse(err)
	}

	t, err := task.Load(sb.store, r, row0.ID)
	if err != nil {
		return refuse(fmt.Errorf("load task %s: %w", row0.ID, err))
	}

	// The correction goes into the record before the run starts, so that a
	// reader looking at why this attempt differed from the last one finds
	// the instruction in the task's own history rather than in a chat log
	// nobody kept.
	corrective := strings.TrimSpace(stringArg(args, "corrective_prompt"))
	if corrective != "" {
		if _, err := verb.Run(sn.context(), sn.world(sb), "note", verb.In{
			Task: t.ID,
			Repo: r.Path,
			Args: map[string]string{"text": supervisorNote(corrective)},
			By:   journalBy,
		}); err != nil {
			return refuse(fmt.Errorf("record the correction on task %s: %w", t.ID, err))
		}
	}

	flow := stringArg(args, "flow")

	out, err := verb.Run(sn.context(), sn.world(sb), "run", verb.In{
		Task: t.ID,
		Repo: r.Path,
		Args: map[string]string{"flow": flow},
		By:   journalBy,
	})
	if err != nil {
		return refuse(fmt.Errorf("start task %s: %w", t.ID, err))
	}

	_ = out

	if flow == "" {
		flow = t.Flow
	}

	// The pid is what the caller watches for: a start that will not say
	// which process it started is a run nobody can point at.
	pid := out.Pid

	trace := journal(sb.store, t, "a model started this task again over mcp, on the %s flow%s", flow, correction(corrective))

	return reply(map[string]any{
		"id":                t.ID,
		"repo":              r.Name,
		"flow":              flow,
		"pid":               pid,
		"attempt":           row0.Attempt + 1,
		"corrective_prompt": corrective != "",
		"message":           fmt.Sprintf("task %s is running again on the %s flow%s", t.ID, flow, trace),
	})
}

// supervisorNote marks a note as having come from a supervising model rather
// than from the person at the keyboard. The record does not carry an author,
// and a directive that reads as though a human wrote it is evidence about
// the wrong party.
func supervisorNote(text string) string {
	return "[supervisor] " + text
}

// correction says whether a restart carried an instruction with it, for the
// line the record keeps. The instruction itself is already a note of its
// own two events earlier; repeating it here would put the same paragraph in
// the record twice.
func correction(corrective string) string {
	if corrective == "" {
		return ""
	}

	return ", after leaving it a correction"
}
