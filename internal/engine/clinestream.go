package engine

import (
	"encoding/json"
	"io"
	"strings"
)

// clineEvent is one line of `cline --json -v`.
//
// Read off cline 3.0.62's own writer, since its published reference still
// describes the say/ask lines of the cline before it:
//
//	{"ts":"…","type":"run_start","providerId":"cline-pass","modelId":"…","sessionId":"1789242437805_ejudv"}
//	{"ts":"…","type":"agent_event","event":{"type":"content_end","contentType":"text","text":"…"}}
//	{"ts":"…","type":"agent_event","event":{"type":"content_start","contentType":"tool","toolName":"editor","input":{…}}}
//	{"ts":"…","type":"agent_event","event":{"type":"content_end","contentType":"tool","toolName":"run_commands","error":"…"}}
//	{"ts":"…","type":"agent_event","event":{"type":"usage","inputTokens":1200,"outputTokens":80,"cost":0.004,…}}
//	{"ts":"…","type":"run_result","finishReason":"completed","text":"Done.","usage":{…}}
//	{"ts":"…","type":"agent_event","event":{"type":"error","error":{"message":"…"},"errorClass":"auth"}}
//
// cline spells the session id sessionId, and writes it on run_start alone.
type clineEvent struct {
	Type    string     `json:"type"`
	Session string     `json:"sessionId"`
	Finish  string     `json:"finishReason"`
	Text    string     `json:"text"`
	Message string     `json:"message"`
	Event   clineInner `json:"event"`
}

// clineInner is the agent_event an agent_event line carries.
type clineInner struct {
	Type        string          `json:"type"`
	ContentType string          `json:"contentType"`
	Text        string          `json:"text"`
	Reasoning   string          `json:"reasoning"`
	ToolName    string          `json:"toolName"`
	Input       json.RawMessage `json:"input"`
	Error       json.RawMessage `json:"error"`

	InputTokens      int64   `json:"inputTokens"`
	OutputTokens     int64   `json:"outputTokens"`
	CacheReadTokens  int64   `json:"cacheReadTokens"`
	CacheWriteTokens int64   `json:"cacheWriteTokens"`
	Cost             float64 `json:"cost"`
}

// usage is one turn's count. cline writes a running total beside it on the
// same line, and the per-turn figures are the ones summed here, the way
// opencode's steps are.
func (e clineInner) usage() Usage {
	return Usage{
		Input:      e.InputTokens,
		Output:     e.OutputTokens,
		CacheRead:  e.CacheReadTokens,
		CacheWrite: e.CacheWriteTokens,
	}
}

// said is what an error field says. A tool's error is a sentence; a run's
// is an object with a message in it.
func (e clineInner) said() string {
	var text string
	if json.Unmarshal(e.Error, &text) == nil {
		return text
	}

	var obj struct {
		Message string `json:"message"`
	}

	if json.Unmarshal(e.Error, &obj) == nil {
		return obj.Message
	}

	return ""
}

// ParseClineStream reads cline's JSON events and reports what the run did.
//
// The answer is the messages cline finished, not its deltas: text arrives
// word by word as content_start and whole as content_end, and only the
// second is read. A tool is recorded when it starts, which is when its input
// is known; one that ends with an error is a refusal when cline refused it.
func ParseClineStream(r io.Reader, onEvent func(StreamEvent)) (Result, error) {
	var (
		out    Result
		texts  []string
		failed string
		found  bool
	)

	lines, err := scanJSONLines(r, func(line []byte) {
		var ev clineEvent
		if json.Unmarshal(line, &ev) != nil {
			return
		}

		switch ev.Type {
		case "run_start":
			found = true
			out.SessionID = ev.Session
		case "run_result":
			found = true

			if ev.Finish != "completed" {
				failed = firstNonEmpty(failed, ev.Text, ev.Finish)
			}
		case "error":
			found = true
			failed = firstNonEmpty(failed, ev.Message)
		case "agent_event":
			found = true

			clineAgentEvent(ev.Event, &out, &texts, &failed, onEvent)
		}
	})
	if err != nil {
		return out, err
	}

	// Said after whatever the run said before it, and never dropped for it:
	// a run that wrote a sentence and then hit its limit is a run that hit
	// its limit, and RanOut reads the answer for the words that say so.
	if failed != "" {
		texts = append(texts, failed)
	}

	out.Output = strings.TrimSpace(strings.Join(texts, "\n\n"))

	if !found {
		return out, silentStream("cline", lines)
	}

	return out, nil
}

// clineAgentEvent reads one agent_event into the result.
func clineAgentEvent(
	e clineInner, out *Result, texts *[]string, failed *string, onEvent func(StreamEvent),
) {
	switch {
	case e.Type == "content_end" && e.ContentType == "text":
		if t := strings.TrimSpace(e.Text); t != "" {
			*texts = append(*texts, t)
		}
	case e.Type == "content_end" && e.ContentType == "reasoning":
		if e.Reasoning != "" {
			out.Thoughts = append(out.Thoughts, e.Reasoning)
			emit(onEvent, StreamEvent{Type: "thought", Thought: e.Reasoning})
		}
	case e.Type == "content_start" && e.ContentType == "tool":
		call := StreamToolCall{Name: e.ToolName, Args: string(e.Input)}
		out.ToolCalls = append(out.ToolCalls, call)
		emit(onEvent, StreamEvent{Type: "tool_call", ToolCall: call})
	case e.Type == "content_end" && e.ContentType == "tool":
		if said := e.said(); strings.Contains(said, "requires approval") {
			refused := StreamRefusal{Tool: e.ToolName, Input: said}
			out.Refusals = append(out.Refusals, refused)
			emit(onEvent, StreamEvent{Type: "refusal", Refusal: refused})
		}
	case e.Type == "usage":
		out.Cost += e.Cost
		out.Usage = addUsage(out.Usage, e.usage())
		emit(onEvent, StreamEvent{Type: "result", Cost: e.Cost})
	case e.Type == "error":
		*failed = firstNonEmpty(e.said(), *failed)
	}
}
