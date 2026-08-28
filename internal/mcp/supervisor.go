package mcp

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/task"
)

// supervisorSay records a message or directive into the persistent supervisor conversation thread.
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

// supervisorHistory reads the persistent supervisor conversation thread.
func (sn Session) supervisorHistory(args map[string]any) CallToolResult {
	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

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
