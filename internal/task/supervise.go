package task

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

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

	fullPrompt := buildSupervisorPrompt(history(events), prompt)
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

// maxHistory is how much of the thread is put in front of the model.
//
// The thread is global, append-only, and nothing prunes it. Without a
// ceiling every call carries every call before it: the prompt grows without
// bound, the bill grows with it, and past some length the engine refuses the
// request outright — so the supervisor would stop answering at all, and the
// reason would be a number nobody was watching.
const maxHistory = 32 << 10

// history is the thread as the model is shown it: the most recent turns that
// fit, oldest first, and a line saying how many were left out.
//
// The most recent rather than the first, because the turn being answered is
// a reply to the last ones. Saying how many were dropped is the same rule
// captured follows for an engine's output: truncation that announces itself
// is honest, and silent loss is not — a model that cannot see it is missing
// context will speak as though it has all of it.
func history(events []record.Event) string {
	// A turn somebody took back is not repeated to the model, and neither
	// is the line that took it back: the model is being shown a
	// conversation, and "that message is withdrawn" is bookkeeping about the
	// conversation rather than part of it. Both are still in the log, which
	// is where a person goes to see what was said.
	gone := record.Retracted(events)

	lines := make([]string, 0, len(events))
	for _, e := range events {
		if e.Kind == record.SupervisorRetracted || gone[record.Stamp(e.At)] {
			continue
		}

		lines = append(lines, historyLine(e))
	}

	kept, budget := 0, maxHistory
	for i := len(lines) - 1; i >= 0; i-- {
		if budget -= len(lines[i]); budget < 0 {
			break
		}

		kept++
	}
	// At least the newest turn, whatever it weighs. One answer longer than
	// the whole budget left kept at zero, and the model was then shown a
	// history made entirely of the line saying how many turns it was not
	// being shown — the turn it is replying to included. A supervisor that
	// writes one long answer blinds itself to the conversation on the very
	// next call. Cutting that turn down is worse than showing it whole and
	// much better than showing nothing.
	if kept == 0 && len(lines) > 0 {
		kept = 1
		lines[len(lines)-1] = trimmed(lines[len(lines)-1], maxHistory)
	}

	var b strings.Builder
	if dropped := len(lines) - kept; dropped > 0 {
		fmt.Fprintf(&b, "…[%d earlier turns are not shown; the thread is longer than this]\n", dropped)
	}

	for _, l := range lines[len(lines)-kept:] {
		b.WriteString(l)
	}

	return b.String()
}

// historyLine is one turn as the model reads it. Who said it and through
// which door are part of the turn: an instruction from the operator and a
// note the supervisor left itself are not the same kind of sentence.
func historyLine(e record.Event) string {
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

	return fmt.Sprintf("[%s via %s]%s: %s\n", by, channel, taskID, e.Text)
}

// trimmed cuts one turn down to n bytes and says in the turn that it had to,
// which is the rule captured follows for an engine's output: truncation that
// announces itself is honest, and silent loss is not.
func trimmed(line string, n int) string {
	if len(line) <= n {
		return line
	}
	// Never sever a rune: this goes into a prompt, and half a character
	// reads to the model as a character it does not know.
	for n > 0 && !utf8.RuneStart(line[n]) {
		n--
	}

	return line[:n] + fmt.Sprintf("…[this turn is %d bytes and is cut here]\n", len(line))
}
