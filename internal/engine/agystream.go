package engine

// Reading agy's stream: one NDJSON event per line, each one an envelope
// naming which of three shapes is inside it.

import (
	"encoding/json"
	"io"
	"strings"
)

// agyEvent is one line of `agy --print --output-format stream-json`.
//
// Captured off a real run rather than guessed:
//
//	{"event":"init","conversation_id":"4eee6f79…","init":{"cwd":"/tmp",…}}
//	{"event":"step_update","step_update":{"step_index":1,"state":"DONE",
//	  "step_type":"agent_response","text_delta":"ok\n","usage":{…}}}
//	{"event":"step_update","step_update":{"step_index":2,"state":"ACTIVE",
//	  "step_type":"tool","tool_name":"list_dir","tool_info":{"name":"list_dir",
//	  "parameters":{"DirectoryPath":"/tmp"}}}}
//	{"event":"step_update","step_update":{"step_index":2,"state":"ERROR",
//	  "step_type":"tool","tool_name":"run_command","tool_info":{…,"error":
//	  {"type":"TOOL_ERROR","message":"permission check failed …"}}}}
//	{"event":"result","result":{"conversation_id":"4eee6f79…",
//	  "status":"SUCCESS","response":"ok\n","num_turns":1,"usage":{…}}}
//
// agy spells the session id conversation_id, which is the fourth spelling of
// that field across four engines and the reason each of them reads its own
// stream.
type agyEvent struct {
	Event string `json:"event"`
	Init  struct {
		Conversation string `json:"conversation_id"`
	} `json:"init"`
	Step struct {
		Conversation string    `json:"conversation_id"`
		Index        int       `json:"step_index"`
		State        string    `json:"state"`
		Type         string    `json:"step_type"`
		Text         string    `json:"text_delta"`
		Tool         string    `json:"tool_name"`
		Usage        *agyUsage `json:"usage"`
		Info         struct {
			Parameters json.RawMessage `json:"parameters"`
			Error      struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"tool_info"`
	} `json:"step_update"`
	Result struct {
		Conversation string    `json:"conversation_id"`
		Status       string    `json:"status"`
		Response     string    `json:"response"`
		Usage        *agyUsage `json:"usage"`
	} `json:"result"`

	// Conversation is the id where an event carries it at the top level.
	Conversation string `json:"conversation_id"`
}

// agyUsage is one count, spelled as agy spells it. thinking_tokens has no
// home in Usage and is dropped: it is a share of the output rather than a
// fourth kind, and the total already carries it.
type agyUsage struct {
	Input     int64 `json:"input_tokens"`
	Output    int64 `json:"output_tokens"`
	CacheRead int64 `json:"cache_read_tokens"`
}

func (u *agyUsage) usage() Usage {
	if u == nil {
		return Usage{}
	}

	return Usage{Input: u.Input, Output: u.Output, CacheRead: u.CacheRead}
}

// ParseAgyStream reads agy's events and reports what the run did.
//
// The result line carries the run's totals rather than its last step's, so
// the counts are taken from there when it arrives and summed off the steps
// when it does not — a run that was killed or cancelled still spent what it
// spent, and there is no third place to read it from.
//
// Cost stays zero. agy reports tokens and no price on any line, which is a
// fact about the engine and not a missing field: what a Google AI Pro
// subscription is being charged is not on this stream.
func ParseAgyStream(r io.Reader, onEvent func(StreamEvent)) (Result, error) {
	var (
		out      Result
		streamed []string
		answer   string
		stepped  Usage
		total    Usage
		found    bool
	)

	lines, err := scanJSONLines(r, func(line []byte) {
		var ev agyEvent
		if json.Unmarshal(line, &ev) != nil {
			return
		}

		if id := firstNonEmpty(ev.Init.Conversation, ev.Step.Conversation, ev.Result.Conversation, ev.Conversation); id != "" {
			out.SessionID = id
		}

		switch ev.Event {
		case "step_update":
			found = true
			stepped = addUsage(stepped, ev.Step.Usage.usage())
			streamed = append(streamed, agyStep(&out, ev, onEvent)...)
		case "result":
			found = true
			total = ev.Result.Usage.usage()
			answer = strings.TrimSpace(ev.Result.Response)

			emit(onEvent, StreamEvent{Type: "result"})
		}
	})
	if err != nil {
		return out, err
	}

	out.Usage = total
	if !total.Any() {
		out.Usage = stepped
	}

	// The result line repeats the whole answer the steps arrived in pieces
	// of, so it is the fallback rather than the last paragraph: added to
	// what was streamed it would write every answer down twice.
	out.Output = strings.TrimSpace(strings.Join(streamed, "\n\n"))
	if out.Output == "" {
		out.Output = answer
	}

	if !found {
		return out, silentStream("agy", lines)
	}

	return out, nil
}

// agyStep reads one step and answers with whatever of it is the run's own
// words.
//
// A step arrives more than once — ACTIVE while it runs, DONE or ERROR when
// it stops — so a tool is recorded on the first of those and the rest are
// read for how it ended. Recording it on every state would draw one command
// in the timeline twice.
func agyStep(out *Result, ev agyEvent, onEvent func(StreamEvent)) []string {
	step := ev.Step

	if step.Type == "tool" && step.State == "ACTIVE" {
		call := StreamToolCall{Name: step.Tool, Args: string(step.Info.Parameters)}
		out.ToolCalls = append(out.ToolCalls, call)
		emit(onEvent, StreamEvent{Type: "tool_call", ToolCall: call})

		return nil
	}

	if step.Type == "tool" && step.State == "ERROR" {
		if isPermissionRefusal(step.Info.Error.Message) {
			ref := StreamRefusal{Tool: step.Tool, Input: string(step.Info.Parameters)}
			out.Refusals = append(out.Refusals, ref)
			emit(onEvent, StreamEvent{Type: "refusal", Refusal: ref})

			return nil
		}

		if msg := strings.TrimSpace(step.Info.Error.Message); msg != "" {
			return []string{msg}
		}

		return nil
	}

	// The answer arrives in pieces on the agent_response steps, and whole
	// again on the result line that ParseAgyStream keeps in reserve.
	if step.Type == "agent_response" {
		if text := strings.TrimSpace(step.Text); text != "" {
			return []string{text}
		}
	}

	return nil
}
