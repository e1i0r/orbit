package engine

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// maxStreamLine is the longest single line ParseStream will read.
//
// One line of claude's stream is one message, and a message carrying a whole
// file can be large; a scanner with the default 64KiB limit would stop part
// way through a run and report a truncated stream as a broken one. Four
// mebibytes is the same order as the record's own line limit, which is what
// bounds the event this ends up in, so nothing survives here that the log
// could not hold anyway. It is a constant rather than a borrowed one because
// internal/engine imports nothing of Orbit's.
const maxStreamLine = 4 << 20

// terminalResult is the object claude prints last, and the only one this
// package reads.
//
// Cost is a plain float64 rather than a pointer: a result object that omits
// the number and one that reports zero are the same fact to a reader of the
// record, which Result's own doc comment already states — an engine that
// does not report a cost is a fact about that engine, not a failure.
type streamEnvelope struct {
	Type      string         `json:"type"`
	Subtype   string         `json:"subtype"`
	Result    string         `json:"result"`
	SessionID string         `json:"session_id"`
	Cost      float64        `json:"total_cost_usd"`
	Message   *streamMessage `json:"message"`
}

type streamMessage struct {
	Content []streamContent `json:"content"`
}

type streamContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	Thinking string          `json:"thinking"`
	Name     string          `json:"name"`
	Input    json.RawMessage `json:"input"`
	Content  string          `json:"content"`
	IsError  bool            `json:"is_error"`
}

// ParseStream reads claude's streaming JSON and returns what the record
// keeps: the human-readable answer, session id, cost, thinking blocks,
// tool calls, and permission refusals.
func ParseStream(r io.Reader) (Result, error) {
	return ParseStreamWithCallback(r, nil)
}

// ParseStreamWithCallback reads claude's streaming JSON, invokes onEvent on each
// incremental event, and returns the aggregated Result.
func ParseStreamWithCallback(r io.Reader, onEvent func(StreamEvent)) (Result, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), maxStreamLine)

	var out Result
	var lines int
	var found bool
	for sc.Scan() {
		lines++
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var env streamEnvelope
		if err := json.Unmarshal(line, &env); err != nil {
			continue
		}

		switch env.Type {
		case "result":
			found = true
			out.Output = env.Result
			out.SessionID = env.SessionID
			out.Cost = env.Cost
			if onEvent != nil {
				onEvent(StreamEvent{Type: "result", Cost: env.Cost})
			}
		case "assistant":
			if env.Message != nil {
				for _, block := range env.Message.Content {
					switch block.Type {
					case "thinking":
						t := block.Thinking
						if t == "" {
							t = block.Text
						}
						if t != "" {
							out.Thoughts = append(out.Thoughts, t)
							if onEvent != nil {
								onEvent(StreamEvent{Type: "thought", Thought: t})
							}
						}
					case "tool_use":
						tc := StreamToolCall{
							Name: block.Name,
							Args: string(block.Input),
						}
						out.ToolCalls = append(out.ToolCalls, tc)
						if onEvent != nil {
							onEvent(StreamEvent{Type: "tool_call", ToolCall: tc})
						}
					}
				}
			}
		case "user":
			if env.Message != nil {
				for _, block := range env.Message.Content {
					if block.Type == "tool_result" && block.IsError {
						if isPermissionRefusal(block.Content) {
							ref := StreamRefusal{
								Tool:  block.Name,
								Input: block.Content,
							}
							out.Refusals = append(out.Refusals, ref)
							if onEvent != nil {
								onEvent(StreamEvent{Type: "refusal", Refusal: ref})
							}
						}
					}
				}
			}
		case "refusal", "permission_denied":
			ref := StreamRefusal{
				Tool:  env.Subtype,
				Input: env.Result,
			}
			out.Refusals = append(out.Refusals, ref)
			if onEvent != nil {
				onEvent(StreamEvent{Type: "refusal", Refusal: ref})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return Result{}, fmt.Errorf("reading the engine's stream after %d lines: %w", lines, err)
	}
	if !found {
		return Result{}, fmt.Errorf("the engine's stream ended after %d lines with no result object: the session id and the cost are reported only there, so this phase has no answer, nothing to resume from and no price", lines)
	}
	return out, nil
}

func isPermissionRefusal(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "not permitted") ||
		strings.Contains(lower, "permission refused") ||
		strings.Contains(lower, "refused") ||
		strings.Contains(lower, "is not allowed")
}
