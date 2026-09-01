package engine

import (
	"encoding/json"
	"io"
	"strings"
)

// codexEvent is one line of `codex exec --json`.
//
// The field names are the ones a real run prints, captured off the binary
// rather than guessed:
//
//	{"type":"thread.started","thread_id":"01a04a53-…"}
//	{"type":"turn.started"}
//	{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"ok"}}
//	{"type":"turn.completed","usage":{"input_tokens":15163,…}}
//	{"type":"error","message":"…"}
//	{"type":"turn.failed","error":{"message":"…"}}
//
// The session id is thread_id. Claude spells the same thing session_id and
// opencode spells it sessionID, which is why each engine parses its own
// stream instead of sharing one: pointed at claude's parser, every line
// above is noise, and that is what made every codex run record no session.
type codexEvent struct {
	Type   string `json:"type"`
	Thread string `json:"thread_id"`
	Msg    string `json:"message"`
	Item   struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Command string `json:"command"`
		Output  string `json:"aggregated_output"`
	} `json:"item"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
	Usage struct {
		Input  int64 `json:"input_tokens"`
		Cached int64 `json:"cached_input_tokens"`
		Output int64 `json:"output_tokens"`
	} `json:"usage"`
}

// usage is codex's count of the turn, in this package's terms.
//
// codex counts the cached prefix inside input_tokens and then names the part
// of it that came from the cache; claude counts the two apart. Subtracting
// here is what keeps Input meaning one thing whichever engine ran the phase
// — added up across a task otherwise, a cached prefix is counted once as
// input and once again as a cache read. The floor is for a codex that
// changes its mind about this: the wrong answer to give then is a zero.
func (e codexEvent) usage() Usage {
	return Usage{
		Input:     max(e.Usage.Input-e.Usage.Cached, 0),
		Output:    e.Usage.Output,
		CacheRead: e.Usage.Cached,
	}
}

// ParseCodexStream reads codex's JSONL and reports what the run did.
//
// Cost is left at zero, and that is an answer rather than a gap: codex
// reports token counts and no price, so there is no figure here this package
// could state without inventing an exchange rate for a model it was not told
// the name of. Result says an empty Cost is a fact about the engine.
func ParseCodexStream(r io.Reader, onEvent func(StreamEvent)) (Result, error) {
	var (
		out      Result
		messages []string
		found    bool
	)

	lines, err := scanJSONLines(r, func(line []byte) {
		var ev codexEvent
		if json.Unmarshal(line, &ev) != nil {
			return
		}

		switch ev.Type {
		case "turn.completed":
			found = true
			out.Usage = ev.usage()
		case "thread.started":
			if ev.Thread != "" {
				out.SessionID = ev.Thread
			}
		case "item.completed":
			found = true
			messages = codexItem(&out, ev, messages, onEvent)
		case "error", "turn.failed":
			found = true

			if msg := firstNonEmpty(ev.Msg, ev.Error.Message); msg != "" {
				messages = append(messages, msg)
			}
		}
	})
	if err != nil {
		return out, err
	}

	out.Output = strings.TrimSpace(strings.Join(messages, "\n\n"))

	if !found {
		return out, silentStream("codex", lines)
	}

	return out, nil
}

// codexItem folds one completed item into the result.
//
// An agent_message is what the phase actually said; reasoning is what it
// thought on the way, which the window shows separately; a command_execution
// is a shell command, which is a tool call by another name. Anything else is
// left alone rather than guessed at — an item type this does not know is not
// a parse failure, it is codex having grown one this build has not met.
func codexItem(out *Result, ev codexEvent, messages []string, onEvent func(StreamEvent)) []string {
	switch ev.Item.Type {
	case "agent_message":
		if ev.Item.Text != "" {
			messages = append(messages, ev.Item.Text)
		}
	case "reasoning":
		if ev.Item.Text != "" {
			out.Thoughts = append(out.Thoughts, ev.Item.Text)
			emit(onEvent, StreamEvent{Type: "thought", Thought: ev.Item.Text})
		}
	case "command_execution":
		call := StreamToolCall{Name: "shell", Args: ev.Item.Command}
		out.ToolCalls = append(out.ToolCalls, call)
		emit(onEvent, StreamEvent{Type: "tool_call", ToolCall: call})
	}

	return messages
}
