package mcp

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/task"
)

// supervisorSay writes one line into the supervisor thread, which is the one
// conversation in Orbit that belongs to no task.
//
// Who is speaking and where from are the caller's to state, and default to a
// supervisor speaking over mcp. That is not a hole: this tool is only ever
// reached over mcp, and every other caller of task.RecordSupervisor — the
// cockpit and the command line both — names itself the same way, so a thread
// whose lines all claimed to come from here would be the lie.
func (sn Session) supervisorSay(args map[string]any) CallToolResult {
	message := strings.TrimSpace(stringArg(args, "message"))
	if message == "" {
		return refuse(fmt.Errorf("this tool needs message"))
	}

	by := strings.TrimSpace(stringArg(args, "by"))
	if by == "" {
		by = "supervisor"
	}

	channel := strings.TrimSpace(stringArg(args, "channel"))
	if channel == "" {
		channel = "mcp"
	}

	taskID := strings.TrimSpace(stringArg(args, "task_id"))
	repo := strings.TrimSpace(stringArg(args, "repo"))

	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

	if err := task.RecordSupervisor(s, record.SupervisorMessage, by, channel, taskID, repo, message); err != nil {
		return refuse(fmt.Errorf("record supervisor message: %w", err))
	}

	return reply(map[string]any{
		"by":      by,
		"channel": channel,
		"task_id": taskID,
		"repo":    repo,
		"message": message,
		"status":  "recorded in supervisor thread",
	})
}

// supervisorHistory reads the thread back, oldest first, optionally only the
// last few lines of it — which is what a supervisor asks for when it wants to
// know what it was last told rather than everything it has ever been told.
func (sn Session) supervisorHistory(args map[string]any) CallToolResult {
	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

	// The root is handed over and not read. The supervisor thread is one
	// file under the state root rather than something folded per repository,
	// so this reader answers the same thread whatever it is pointed at — and
	// passing the session's root is still what to do, because the day that
	// stops being true the honest value is already here.
	r := board.NewReader(s, sn.Root)

	lines, err := r.SupervisorLog()
	if err != nil {
		return refuse(fmt.Errorf("read supervisor history: %w", err))
	}

	limit := 0
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	if limit > 0 && len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}

	return reply(map[string]any{
		"count":    len(lines),
		"messages": lines,
	})
}
