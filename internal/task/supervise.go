package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Supervise invokes a model to act as the cockpit's supervisor.
//
// The prompt is augmented with context about Orbit's MCP tools and supervisor role.
// The model's answer is recorded in supervisor.jsonl and returned.
func Supervise(ctx context.Context, s *store.Store, eng engine.Engine, prompt string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("store cannot be nil")
	}
	if eng == nil {
		return "", fmt.Errorf("engine cannot be nil")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("supervise prompt cannot be empty")
	}

	events, err := SupervisorEvents(s)
	if err != nil {
		events = nil
	}
	var historyBuilder strings.Builder
	for _, e := range events {
		by := e.Data["by"]
		if by == "" {
			by = "operator"
		}
		channel := e.Data["channel"]
		if channel == "" {
			channel = "tui"
		}
		taskID := ""
		if e.Data["task_id"] != "" {
			taskID = " (" + e.Data["task_id"] + ")"
		}
		fmt.Fprintf(&historyBuilder, "[%s via %s]%s: %s\n", by, channel, taskID, e.Text)
	}

	fullPrompt := buildSupervisorPrompt(historyBuilder.String(), prompt)
	req := engine.Request{
		Prompt:      fullPrompt,
		Dir:         s.Root(),
		Permissions: []string{engine.PermissionRead, engine.PermissionRepo, engine.PermissionNetwork},
	}

	out, runErr := eng.Run(ctx, req)
	ans := strings.TrimSpace(out.Output)
	if runErr != nil && ans == "" {
		return "", fmt.Errorf("supervisor engine %s failed: %w", eng.Name(), runErr)
	}

	if ans != "" {
		if recErr := RecordSupervisor(s, record.SupervisorMessage, eng.Name(), "supervisor", "", "", ans); recErr != nil {
			return ans, recErr
		}
	}
	return ans, nil
}

// AutoSupervise triggers the supervisor autonomously when autopilot is on and
// tasks require inspection or remediation.
func AutoSupervise(ctx context.Context, s *store.Store, eng engine.Engine, needingAttention []string) (string, error) {
	prompt := fmt.Sprintf("Autopilot is active. The following tasks require your inspection: %s. Inspect their records (orbit_inspect_task), analyze any errors or gates, direct or retry them if appropriate, and post a concise debriefing.", strings.Join(needingAttention, ", "))
	return Supervise(ctx, s, eng, prompt)
}

func buildSupervisorPrompt(history, newPrompt string) string {
	var b strings.Builder
	b.WriteString("You are Orbit's Supervisor sitting at the cockpit seat.\n")
	b.WriteString("You have full authority to inspect all repositories, tasks, and flows via Orbit MCP tools.\n")
	b.WriteString("Answer the operator directly, take corrective actions on tasks when needed, and report clearly.\n\n")
	if history != "" {
		b.WriteString("## Supervisor Thread History:\n")
		b.WriteString(history)
		b.WriteString("\n")
	}
	b.WriteString("## Operator Directive / Message:\n")
	b.WriteString(newPrompt)
	b.WriteString("\n\nRespond concisely with your assessment, conclusions, or actions taken.")
	return b.String()
}
