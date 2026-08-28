package mcp

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
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

	r, err := pickRepo(sb.board, stringArg(args, "repo"))
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

	t, err := task.Create(sb.store, r, id, text, stringArg(args, "flow"))
	if err != nil {
		return refuse(fmt.Errorf("write task %s down in %s: %w", id, r.Name, err))
	}

	trace := journal(sb.store, t, "a model wrote this task down over mcp; nobody has started it")

	return reply(map[string]any{
		"id":        t.ID,
		"repo":      r.Name,
		"repo_path": r.Path,
		"flow":      t.Flow,
		"band":      bandSlug(view.ToDo),
		"started":   false,
		"message":   fmt.Sprintf("task %s is written against %s and will walk the %s flow; start it with orbit_retry_task%s", t.ID, r.Name, t.Flow, trace),
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
// It is task.Start and not the resume control word, which is what the first
// version of this tool wrote. Those are different acts: resume lets a live
// run past a gate, and a run that failed an hour ago has no process to let
// past — the word would sit in the control file unread while the tool
// reported success.
func (sn Session) retryTask(args map[string]any) CallToolResult {
	sb, err := sn.readBoard()
	if err != nil {
		return refuse(err)
	}

	row0, err := findTask(sb.board, stringArg(args, "task_id"), stringArg(args, "repo"))
	if err != nil {
		return refuse(err)
	}

	if row0.Live {
		return refuse(fmt.Errorf("task %s is running; pause it with orbit_pause_task or stop it with orbit_cancel_task before starting it again", row0.ID))
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
		if err := task.Note(sb.store, t, supervisorNote(corrective)); err != nil {
			return refuse(fmt.Errorf("record the correction on task %s: %w", t.ID, err))
		}
	}

	chosen := stringArg(args, "flow")
	if chosen == "" {
		chosen = t.Flow
	}

	pid, err := task.Start(sb.store, t, chosen, board.Unread(sb.board))
	if err != nil {
		return refuse(fmt.Errorf("start task %s: %w", t.ID, err))
	}

	trace := journal(sb.store, t, "a model started this task again over mcp, on the %s flow%s", chosen, correction(corrective))

	return reply(map[string]any{
		"id":                t.ID,
		"repo":              r.Name,
		"flow":              chosen,
		"pid":               pid,
		"attempt":           row0.Attempt + 1,
		"corrective_prompt": corrective != "",
		"message":           fmt.Sprintf("task %s is running again on the %s flow%s", t.ID, chosen, trace),
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

// addNote writes a note into a task's record, where the cockpit's notes tab
// reads it.
func (sn Session) addNote(args map[string]any) CallToolResult {
	text := strings.TrimSpace(stringArg(args, "text"))
	if text == "" {
		return refuse(fmt.Errorf("this tool needs text"))
	}

	sb, t, res := sn.loadFor(args)
	if res != nil {
		return *res
	}

	note := supervisorNote(text)
	if err := task.Note(sb.store, t, note); err != nil {
		return refuse(fmt.Errorf("record a note on task %s: %w", t.ID, err))
	}

	return done("noted on task %s: %s", t.ID, note)
}

// control leaves one of the run's control words for a task, and says so in
// the past tense the caller can repeat back.
func (sn Session) control(args map[string]any, word, past string) CallToolResult {
	sb, t, res := sn.loadFor(args)
	if res != nil {
		return *res
	}

	if err := task.Control(sb.store, t, word); err != nil {
		return refuse(fmt.Errorf("tell task %s to %s: %w", t.ID, word, err))
	}

	trace := journal(sb.store, t, "a model asked this task to %s over mcp", word)
	// The word is on disk and the run reads it at its next phase boundary,
	// which has not happened yet. Saying it is paused would be a claim about
	// something this process cannot see.
	return done("task %s will be %s at its next phase boundary; orbit_inspect_task says whether it has been%s", t.ID, past, trace)
}

// cancelTask stops a run where it stands.
func (sn Session) cancelTask(args map[string]any) CallToolResult {
	sb, t, res := sn.loadFor(args)
	if res != nil {
		return *res
	}

	if err := task.Cancel(sb.store, t); err != nil {
		return refuse(fmt.Errorf("cancel task %s: %w", t.ID, err))
	}

	trace := journal(sb.store, t, "a model cancelled this task over mcp")

	return done("task %s was told to stop; the record carries the outcome%s", t.ID, trace)
}

// directTask interrupts an in-flight run while preserving memory, records the
// directive and note, and optionally restarts the task.
func (sn Session) directTask(args map[string]any) CallToolResult {
	message := strings.TrimSpace(stringArg(args, "message"))
	if message == "" {
		return refuse(fmt.Errorf("this tool needs message"))
	}

	sb, t, res := sn.loadFor(args)
	if res != nil {
		return *res
	}
	// Directing and restarting is one verb — task.Reopen — and not the two
	// halves of it done here in a row. Reopen waits for the run it stopped
	// to actually be gone before starting the next; spelling the pair out a
	// second time is how this path came to skip that wait and put two runs
	// on one task.
	if boolArg(args, "restart") {
		pid, err := task.Reopen(sb.store, t, "mcp", message, t.Flow, board.Unread(sb.board))
		if err != nil {
			return refuse(fmt.Errorf("restart task %s after directing it: %w", t.ID, err))
		}

		trace := journal(sb.store, t, "a model directed this task over mcp and started it again")

		return reply(map[string]any{
			"id":      t.ID,
			"pid":     pid,
			"message": fmt.Sprintf("task %s was directed and restarted with pid %d%s", t.ID, pid, trace),
		})
	}

	if err := task.Direct(sb.store, t, "mcp", message); err != nil {
		return refuse(fmt.Errorf("direct task %s: %w", t.ID, err))
	}

	trace := journal(sb.store, t, "a model directed this task over mcp")

	return done("task %s was directed; the directive is recorded%s", t.ID, trace)
}

// loadFor is the three steps every tool that writes to a task takes: fold
// the board, find the row the caller named, and load the task behind it.
//
// The refusal comes back as a *CallToolResult rather than an error so that
// each caller returns it unchanged; there is no case where one of these
// tools has something to add to why the task could not be found.
func (sn Session) loadFor(args map[string]any) (*storeAndBoard, task.Task, *CallToolResult) {
	sb, err := sn.readBoard()
	if err != nil {
		res := refuse(err)
		return nil, task.Task{}, &res
	}

	row0, err := findTask(sb.board, stringArg(args, "task_id"), stringArg(args, "repo"))
	if err != nil {
		res := refuse(err)
		return nil, task.Task{}, &res
	}

	r, err := openTaskRepo(row0)
	if err != nil {
		res := refuse(err)
		return nil, task.Task{}, &res
	}

	t, err := task.Load(sb.store, r, row0.ID)
	if err != nil {
		res := refuse(fmt.Errorf("load task %s: %w", row0.ID, err))
		return nil, task.Task{}, &res
	}

	return sb, t, nil
}
