package mcp

// The tools that write, as adapters over the declaration.
//
// The names are the ones models learned — orbit_create_task and not
// orbit_new — and the extras are a model's: what it writes is marked as
// coming from a supervisor rather than from the person at the keyboard,
// and what it did is journaled on the task for whoever reads next. What
// writing means is the verb's, asked here the way every other door asks
// it, so a task written by a model and one written at a terminal are the
// same task — refused for the same reasons, recorded with the same words.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/verb"
)

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

	defer sb.close()

	if _, err := verb.Run(sn.context(), sn.world(sb), "task note", verb.In{
		Task: t.ID,
		Args: map[string]string{"text": supervisorNote(text)},
		By:   journalBy,
	}); err != nil {
		return refuse(fmt.Errorf("record a note on task %s: %w", t.ID, err))
	}

	return done("noted on task %s: %s", t.ID, supervisorNote(text))
}

// control leaves one of the run's control words for a task, and says so in
// the past tense the caller can repeat back.
func (sn Session) control(args map[string]any, word, past string) CallToolResult {
	sb, t, res := sn.loadFor(args)
	if res != nil {
		return *res
	}

	defer sb.close()

	out, err := verb.Run(sn.context(), sn.world(sb), "task "+word, verb.In{
		Task: t.ID,
		By:   journalBy,
	})
	if err != nil {
		return refuse(fmt.Errorf("tell task %s to %s: %w", t.ID, word, err))
	}

	_ = out

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

	defer sb.close()

	out, err := verb.Run(sn.context(), sn.world(sb), "task cancel", verb.In{
		Task: t.ID,
		By:   journalBy,
	})
	if err != nil {
		return refuse(fmt.Errorf("cancel task %s: %w", t.ID, err))
	}

	_ = out

	trace := journal(sb.store, t, "a model cancelled this task over mcp")

	return done("task %s was told to stop; the record carries the outcome%s", t.ID, trace)
}

// requeueTask stops whatever holds a task and puts it back in the queue.
func (sn Session) requeueTask(args map[string]any) CallToolResult {
	sb, t, res := sn.loadFor(args)
	if res != nil {
		return *res
	}

	defer sb.close()

	out, err := verb.Run(sn.context(), sn.world(sb), "task requeue", verb.In{
		Task: t.ID,
		Args: map[string]string{"why": strings.TrimSpace(stringArg(args, "why"))},
		By:   journalBy,
	})
	if err != nil {
		return refuse(fmt.Errorf("requeue task %s: %w", t.ID, err))
	}

	_ = out

	trace := journal(sb.store, t, "a model put this task back in the queue over mcp")

	return done("task %s is back in to do and can be started again%s", t.ID, trace)
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

	defer sb.close()

	out, err := verb.Run(sn.context(), sn.world(sb), "task direct", verb.In{
		Task: t.ID,
		Args: map[string]string{
			"text":    message,
			"restart": strings.TrimSpace(stringArg(args, "restart")),
		},
		By: journalBy,
	})
	if err != nil {
		return refuse(fmt.Errorf("direct task %s: %w", t.ID, err))
	}

	_ = out

	if boolArg(args, "restart") {
		trace := journal(sb.store, t, "a model directed this task over mcp and started it again")

		return reply(map[string]any{
			"id":      t.ID,
			"pid":     out.Pid,
			"message": fmt.Sprintf("task %s was directed and restarted with pid %d%s", t.ID, out.Pid, trace),
		})
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

	row0, err := findTask(sb.board, stringArg(args, "task_id"))
	if err != nil {
		sb.close()

		res := refuse(err)

		return nil, task.Task{}, &res
	}

	r, err := openTaskRepo(row0)
	if err != nil {
		sb.close()

		res := refuse(err)

		return nil, task.Task{}, &res
	}

	t, err := task.Load(sb.store, r, row0.ID)
	if err != nil {
		sb.close()

		res := refuse(fmt.Errorf("load task %s: %w", row0.ID, err))

		return nil, task.Task{}, &res
	}
	// The record travels with the row, and so does closing it: the caller
	// owns what it was handed. Every path above that answers instead gives
	// it back here, because a refusal that leaked would be the tool call
	// that costs the most and keeps the most.
	return sb, t, nil
}
