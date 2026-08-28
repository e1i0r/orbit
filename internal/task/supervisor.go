package task

import (
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// RecordSupervisor appends an event to the global supervisor conversation log.
func RecordSupervisor(s *store.Store, kind, by, channel, taskID, repo, text string) error {
	if s == nil {
		return fmt.Errorf("store cannot be nil")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("supervisor message cannot be empty")
	}
	if kind == "" {
		kind = record.SupervisorMessage
	}
	data := map[string]string{}
	if by = strings.TrimSpace(by); by != "" {
		data["by"] = by
	}
	if channel = strings.TrimSpace(channel); channel != "" {
		data["channel"] = channel
	}
	if taskID = strings.TrimSpace(taskID); taskID != "" {
		data["task_id"] = taskID
	}
	if repo = strings.TrimSpace(repo); repo != "" {
		data["repo"] = repo
	}

	e := record.Event{
		At:   time.Now().UTC(),
		Kind: kind,
		Text: text,
		Data: data,
	}
	return record.Append(s.SupervisorLogPath(), e)
}

// SupervisorEvents reads all events from the global supervisor conversation log.
func SupervisorEvents(s *store.Store) ([]record.Event, error) {
	if s == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}
	return record.Read(s.SupervisorLogPath())
}
