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
const maxStreamLine = 4 << 20

type streamEnvelope struct {
	Type         string          `json:"type"`
	Subtype      string          `json:"subtype"`
	Result       string          `json:"result"`
	SessionID    string          `json:"session_id"`
	Cost         float64         `json:"total_cost_usd"`
	Message      *streamMessage  `json:"message"`
	ContentBlock *streamContent  `json:"content_block"`
	Delta        *streamDelta    `json:"delta"`
	Name         string          `json:"name"`
	Input        json.RawMessage `json:"input"`
	Text         string          `json:"text"`
	Thinking     string          `json:"thinking"`
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

type streamDelta struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Thinking string `json:"thinking"`
}

// ParseStream reads claude's streaming JSON and returns what the record keeps.
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
		case "content_block_start":
			if env.ContentBlock != nil {
				switch env.ContentBlock.Type {
				case "thinking":
					th := env.ContentBlock.Thinking
					if th == "" {
						th = env.ContentBlock.Text
					}
					if th != "" {
						out.Thoughts = append(out.Thoughts, th)
						if onEvent != nil {
							onEvent(StreamEvent{Type: "thought", Thought: th})
						}
					}
				case "text":
					if env.ContentBlock.Text != "" {
						out.Thoughts = append(out.Thoughts, env.ContentBlock.Text)
						if onEvent != nil {
							onEvent(StreamEvent{Type: "thought", Thought: env.ContentBlock.Text})
						}
					}
				case "tool_use":
					tc := StreamToolCall{
						Name: env.ContentBlock.Name,
						Args: string(env.ContentBlock.Input),
					}
					out.ToolCalls = append(out.ToolCalls, tc)
					if onEvent != nil {
						onEvent(StreamEvent{Type: "tool_call", ToolCall: tc})
					}
				}
			}
		case "content_block_delta":
			if env.Delta != nil {
				switch env.Delta.Type {
				case "thinking_delta":
					if env.Delta.Thinking != "" && onEvent != nil {
						onEvent(StreamEvent{Type: "thought", Thought: env.Delta.Thinking})
					}
				case "text_delta":
					if env.Delta.Text != "" && onEvent != nil {
						onEvent(StreamEvent{Type: "thought", Thought: env.Delta.Text})
					}
				}
			}
		case "tool_use", "tool_call":
			tc := StreamToolCall{
				Name: env.Name,
				Args: string(env.Input),
			}
			out.ToolCalls = append(out.ToolCalls, tc)
			if onEvent != nil {
				onEvent(StreamEvent{Type: "tool_call", ToolCall: tc})
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
					case "text":
						if block.Text != "" {
							out.Thoughts = append(out.Thoughts, block.Text)
							if onEvent != nil {
								onEvent(StreamEvent{Type: "thought", Thought: block.Text})
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
