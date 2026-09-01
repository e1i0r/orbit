package engine

import (
	"encoding/json"
	"io"
	"strings"
)

// openCodeEvent is one line of `opencode run --format json`.
//
// Captured off a real run rather than guessed:
//
//	{"type":"step_start","sessionID":"ses_fb5a…","part":{"type":"step-start",…}}
//	{"type":"text","sessionID":"ses_fb5a…","part":{"type":"text","text":"ok",…}}
//	{"type":"tool_use","sessionID":"…","part":{"type":"tool","tool":"write",
//	  "callID":"…","state":{"status":"completed","input":{…}}}}
//	{"type":"step_finish","sessionID":"…","part":{"reason":"stop",
//	  "tokens":{…},"cost":0}}
//	{"type":"error","sessionID":"…","error":{"name":"UnknownError","data":{…}}}
//
// opencode spells the session id sessionID, codex spells it thread_id and
// claude spells it session_id — three names for the one field a resume is
// built from, which is the whole reason each engine reads its own stream.
type openCodeEvent struct {
	Type    string `json:"type"`
	Session string `json:"sessionID"`
	Part    struct {
		Type  string  `json:"type"`
		Text  string  `json:"text"`
		Tool  string  `json:"tool"`
		Cost  float64 `json:"cost"`
		State struct {
			Status string          `json:"status"`
			Input  json.RawMessage `json:"input"`
		} `json:"state"`
		Tokens struct {
			Input  int64 `json:"input"`
			Output int64 `json:"output"`
			Cache  struct {
				Read  int64 `json:"read"`
				Write int64 `json:"write"`
			} `json:"cache"`
		} `json:"tokens"`
	} `json:"part"`
	Error struct {
		Name string `json:"name"`
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	} `json:"error"`
}

// usage is one step's count, spelled as opencode spells it: the cache pair
// is nested under the tokens object rather than named beside the other two.
func (e openCodeEvent) usage() Usage {
	t := e.Part.Tokens

	return Usage{
		Input:      t.Input,
		Output:     t.Output,
		CacheRead:  t.Cache.Read,
		CacheWrite: t.Cache.Write,
	}
}

// ParseOpenCodeStream reads opencode's JSON events and reports what the run
// did.
//
// Cost is summed across steps rather than read off one line: opencode prices
// each step as it finishes, and a run that took four steps reports four
// figures. A free model reports zero on every one of them, which is the
// right answer and not a missing one.
func ParseOpenCodeStream(r io.Reader, onEvent func(StreamEvent)) (Result, error) {
	var (
		out   Result
		texts []string
		found bool
	)

	lines, err := scanJSONLines(r, func(line []byte) {
		var ev openCodeEvent
		if json.Unmarshal(line, &ev) != nil {
			return
		}

		if ev.Session != "" {
			out.SessionID = ev.Session
		}

		switch ev.Type {
		case "text":
			found = true

			if ev.Part.Text != "" {
				texts = append(texts, ev.Part.Text)
			}
		case "reasoning":
			found = true

			if ev.Part.Text != "" {
				out.Thoughts = append(out.Thoughts, ev.Part.Text)
				emit(onEvent, StreamEvent{Type: "thought", Thought: ev.Part.Text})
			}
		case "tool_use":
			found = true

			call := StreamToolCall{Name: ev.Part.Tool, Args: string(ev.Part.State.Input)}
			out.ToolCalls = append(out.ToolCalls, call)
			emit(onEvent, StreamEvent{Type: "tool_call", ToolCall: call})
		case "step_finish":
			found = true
			out.Cost += ev.Part.Cost
			// Summed across steps for cost's reason: opencode counts each
			// step as it finishes, so a run of four steps reports four
			// counts and the phase spent all of them.
			out.Usage = addUsage(out.Usage, ev.usage())

			emit(onEvent, StreamEvent{Type: "result", Cost: ev.Part.Cost})
		case "error":
			found = true

			if msg := firstNonEmpty(ev.Error.Data.Message, ev.Error.Name); msg != "" {
				texts = append(texts, msg)
			}
		}
	})
	if err != nil {
		return out, err
	}

	out.Output = strings.TrimSpace(strings.Join(texts, "\n\n"))

	if !found {
		return out, silentStream("opencode", lines)
	}

	return out, nil
}
