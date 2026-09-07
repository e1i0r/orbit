// Package supervisor is the one conversation in Orbit that belongs to no
// task: a global, append-only thread under the state root, and the model
// that answers into it.
//
// It lived in internal/task until this package existed, which made a package
// whose own doc says it turns a written sentence into a run also the home of
// a chat log, and put the whole lifecycle of a run between every reader and
// that log. Nothing here takes a task, holds a marker, or walks a flow. The
// supervisor acts on tasks the way everything else does — through
// internal/cli and internal/mcp — and never from in here, which is what
// keeps the direction one way.
package supervisor

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/knowledge"
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

	conversation, err := Current(s)
	if err != nil {
		return "", err
	}

	return SuperviseIn(ctx, s, eng, conversation, prompt)
}

// SuperviseIn is the same, told which conversation it is in.
//
// What the model is shown is that conversation and no other. The thread used
// to be one list with no ends, so every answer carried every line anybody had
// ever written — including the ones about a repository nobody has touched
// since. What survives between conversations is what Orbit knows, which is
// the point of writing a fact down rather than saying it.
func SuperviseIn(ctx context.Context, s *store.Store, eng engine.Engine, conversation, prompt string) (string, error) {
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

	// A thread that cannot be read is refused rather than answered around.
	// Swallowing the error handed the model an empty history, and history
	// below goes to some length to make sure that never happens quietly: a
	// model that cannot see it is missing context speaks as though it has all
	// of it. This supervisor does not only speak — it directs tasks, retries
	// them and cancels them — so one that cannot remember what it already did
	// is one that does it again. The operator sees the error and can fix the
	// file; a supervisor with no memory is not something they can see at all.
	events, err := Events(s)
	if err != nil {
		return "", fmt.Errorf("read the supervisor thread: %w", err)
	}

	// What the record says, beside the conversation. It is read here and
	// not left to the tools: a supervisor asked "what happened?" that has to
	// go and look answers from whatever it thought to ask for, and a
	// supervisor handed the lines answers from what is written down. The
	// failure to read it is not fatal — the question can still be answered
	// from the thread — but it is said out loud in the block, because a
	// model that cannot see it is missing context speaks as though it has
	// all of it.
	happened, err := Happened(s, time.Now().Add(-happenedWindow))
	if err != nil {
		happened = []string{"the record could not be read: " + err.Error()}
	}

	thread := history(In(conversation, events))
	fullPrompt := buildSupervisorPrompt(thread, happened, prompt, standing(s))
	req := engine.Request{
		Prompt:      fullPrompt,
		Dir:         s.Root(),
		Permissions: []string{engine.PermissionRead, engine.PermissionRepo, engine.PermissionNetwork},
	}

	out, runErr := eng.Run(ctx, req)

	// Whatever it managed to say is kept, the way a phase keeps what its
	// engine printed before it died (phase.go): a half-finished answer is
	// the only account there is of what the supervisor was doing.
	ans := strings.TrimSpace(out.Output)
	if ans != "" {
		if recErr := Record(s, record.SupervisorMessage, eng.Name(), "supervisor", "", "", ans); recErr != nil {
			return ans, recErr
		}
	}

	// And the error still travels. Dropping it whenever the engine printed
	// anything at all would record an answer cut off halfway — by a
	// cancellation, a crash, a quota — in the thread and hand it back as
	// though the supervisor had finished speaking. The cockpit already draws
	// both: the thread it re-reads from this log, and the error on the status
	// line (ui/update.go).
	if runErr != nil {
		return ans, fmt.Errorf("supervisor engine %s failed: %w", eng.Name(), runErr)
	}

	return ans, nil
}

// AutoSupervise triggers the supervisor autonomously when autopilot is on and
// tasks require inspection or remediation.
func AutoSupervise(ctx context.Context, s *store.Store, eng engine.Engine, needingAttention []string) (string, error) {
	prompt := fmt.Sprintf("Autopilot is active. The following tasks require your inspection: %s. Inspect their records (orbit_inspect_task), analyze any errors or gates, direct or retry them if appropriate, and post a concise debriefing.", strings.Join(needingAttention, ", "))
	return Supervise(ctx, s, eng, prompt)
}

// buildSupervisorPrompt is what the supervisor is handed: who it is, the
// thread so far, the message it is answering, and how to answer it.
//
// It is written in Markdown for the reason a phase's prompt is (task/
// run_helpers.go): the answer is asked for in Markdown and drawn as Markdown
// in the cockpit, and a prompt that asks in one shape for another is asking
// twice.
func buildSupervisorPrompt(
	history string, happened []string, newPrompt string, knows []knowledge.Fact,
) string {
	var b strings.Builder

	b.WriteString("# Supervisor\n\n")
	b.WriteString("You are Orbit's supervisor, sitting at the cockpit seat. " +
		"You can inspect every repository, task and flow through Orbit's MCP tools. " +
		"Answer the operator directly, act on the tasks that need acting on, and say what you did.\n")

	// Fenced rather than set as prose: every answer in the thread was
	// written to the same contract this prompt ends with, so its headings
	// loose under a heading of this prompt would read as sections of it.
	if history != "" {
		fmt.Fprintf(&b, "\n## Thread so far\n\n%s\n", engine.Fenced(history))
	}

	// The record itself, not a verdict about it. Everything in this block
	// happened and was written down; everything the supervisor says beyond
	// it is its own reading, and the contract below asks it to keep the two
	// apart — "the tests passed" is a different claim from "it looks like
	// the tests passed", and the person coming back from lunch is deciding
	// what to do next on the strength of which one it is.
	if len(happened) > 0 {
		fmt.Fprintf(&b, "\n## What the record says (the last %d hours, verified)\n\n%s\n",
			int(happenedWindow.Hours()), engine.Fenced(strings.Join(happened, "\n")))
	}

	b.WriteString(alreadyKnown(knows))
	fmt.Fprintf(&b, "\n## Operator message\n\n%s\n", strings.TrimSpace(newPrompt))
	b.WriteString("\n" + answerContract)

	return b.String()
}

// answerContract is how the supervisor is asked to answer, and it is its own
// rather than the one a phase gets.
//
// A phase writes a work report: it ran for minutes with nobody watching, and
// the panes that draw its answer lay out headings and sections. The
// supervisor is a conversation — somebody asked it something and is sitting
// in front of the screen waiting — and handed the phase's contract it turned
// every "what happened?" into a page of headings, bullet lists, fenced
// blocks and file:// links that wrapped mid-path.
//
// So: answer the question, in the length the question deserves. The
// structure is allowed when what was asked for is genuinely a list; it is
// not the default.
const answerContract = "## How to answer\n\n" +
	"You are talking to somebody who is looking at the screen, not writing them a report.\n\n" +
	"- Answer the question asked, in a few sentences. Long only when what was asked for is long.\n" +
	"- No headings unless the answer is genuinely several sections. Prefer a paragraph.\n" +
	"- Bullets only for a list of things. Not for one thing.\n" +
	"- Never write file:// links or any other address: nothing here can be clicked, and a path " +
	"wraps mid-word in the middle of a sentence. Name the file plainly instead.\n" +
	"- Answer in the language the operator wrote in. They asked in it; the record and this prompt " +
	"are in English because that is what the log is written in, and that is not what to answer in.\n" +
	"- Say what you did, if you did something. Do not restate what was asked.\n" +
	"- Asked what happened, answer from the record block above: which tasks ran, how they ended, " +
	"which checks passed and which failed. It is what is written down.\n" +
	"- Keep verified apart from your own reading. A check that exited zero is verified and you may " +
	"say so plainly; anything you conclude beyond the record is yours, and say that it is. " +
	"Never call something verified because it looks right.\n" +
	"- What is missing from the record is missing: say so rather than filling it in. " +
	"orbit_inspect_task reads a task in full when the block is not enough.\n\n" +
	"When the operator tells you something that should have been standing knowledge about the code — " +
	"a constraint, a trap, a convention, something they are telling you because Orbit did not already " +
	"know it — **offer** to write it down with orbit_learn, in one line, at the end of your answer. " +
	"Never write one without them agreeing: a rule that appeared because nobody disagreed is a rule " +
	"nobody put there. Say nothing when what they said is a question, a request, or something already " +
	"in the list above.\n"

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
