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
	Usage        *streamUsage    `json:"usage"`
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
	Usage   *streamUsage    `json:"usage"`
}

// streamUsage is claude's count of one turn, spelled as claude spells it.
type streamUsage struct {
	Input      int64 `json:"input_tokens"`
	Output     int64 `json:"output_tokens"`
	CacheRead  int64 `json:"cache_read_input_tokens"`
	CacheWrite int64 `json:"cache_creation_input_tokens"`
}

func (u *streamUsage) usage() Usage {
	if u == nil {
		return Usage{}
	}

	return Usage{Input: u.Input, Output: u.Output, CacheRead: u.CacheRead, CacheWrite: u.CacheWrite}
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
//
// A thinking block is a thought and a text block is the answer, and the two
// are kept apart the way codexstream.go and opencodestream.go keep them
// apart. Text blocks used to go into Thoughts as well, and the same prose
// they carry comes back on the result line as Output — so every phase wrote
// its answer down twice, once as the thoughts the thinking pane draws and
// once as the text beside "finished".
func ParseStreamWithCallback(r io.Reader, onEvent func(StreamEvent)) (Result, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), maxStreamLine)

	var (
		out        Result
		texts      []string
		lines      int
		found      bool
		perMessage Usage
	)

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

		if env.SessionID != "" {
			out.SessionID = env.SessionID
		}

		// Every assistant message carries its own count, and a stream that
		// ends without a result line — a run killed, a phase cancelled — is
		// still a phase that spent something. Summed here and used only if
		// no result line ever arrives.
		if u := usageOf(env.Message); u.Any() {
			perMessage = addUsage(perMessage, u)
		}

		switch env.Type {
		case "result":
			found = true
			out.Output = env.Result
			// Not out.SessionID = env.SessionID. The guard above already
			// took the id from whichever line carried one, and this line
			// overwrote it with whatever the result object held —
			// including nothing. A result without a session_id wiped the
			// id captured off the init line, which is the only reason
			// claude is asked for stream-json in the first place: the run
			// finished, the record said it could not be resumed, and the
			// evidence that it could had been read and thrown away.
			out.Cost = env.Cost
			// The result line's own count is the turn's total, so it wins
			// over the running sum below rather than adding to it: taking
			// both would count every assistant message twice.
			if u := env.Usage.usage(); u.Any() {
				out.Usage = u
			}

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
						texts = append(texts, env.ContentBlock.Text)
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
			// Thinking deltas only. A text delta is a fragment of the
			// answer, and the answer is taken whole from the block that
			// carries it; a fragment sent on as a thought would be the
			// prose of the report arriving a second time, in pieces.
			if env.Delta != nil && env.Delta.Type == "thinking_delta" &&
				env.Delta.Thinking != "" && onEvent != nil {
				onEvent(StreamEvent{Type: "thought", Thought: env.Delta.Thinking})
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
							texts = append(texts, block.Text)
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

	// The result line is where the answer is when the run reached the end of
	// itself. The text blocks are what is left when it did not: a phase
	// killed mid-run has no result object, and everything it said is in
	// them. Without this, spec.report falls back to the raw stream, and what
	// the record keeps of a cancelled phase is its JSON frames.
	if out.Output == "" {
		out.Output = strings.TrimSpace(strings.Join(texts, "\n\n"))
	}

	if !out.Usage.Any() {
		out.Usage = perMessage
	}

	if !found {
		return out, fmt.Errorf("the engine's stream ended after %d lines with no result object: the session id and the cost are reported only there, so this phase has nothing to resume from and no price", lines)
	}

	return out, nil
}

// isPermissionRefusal is whether a failed tool result is the sandbox saying
// no, rather than the tool itself failing.
//
// Every phrase here names a permission. The bare word "refused" is not on
// the list, because it is the one thing a network stack says when nothing is
// listening: "dial tcp 127.0.0.1:5432: connect: connection refused" matched
// as a phase being denied permission makes a run that failed because a
// database was down read, in the record and on the screen, as a run whose
// posture was too narrow — pointing whoever debugs it at the permissions and
// away from the port.
//
// Matching another program's prose is a guess whichever words are chosen.
// The guess these make is narrow on purpose: a refusal this misses is a
// refusal reported as a plain tool failure, which is the milder of the two
// wrong answers.
func isPermissionRefusal(s string) bool {
	lower := strings.ToLower(s)

	return strings.Contains(lower, "permission denied") ||
		// agy's own wording, which puts who said no in front of it: "user
		// denied permission to run command".
		strings.Contains(lower, "denied permission") ||
		strings.Contains(lower, "not permitted") ||
		strings.Contains(lower, "permission refused") ||
		strings.Contains(lower, "refused permission") ||
		strings.Contains(lower, "is not allowed")
}

// usageOf is one message's count, for a message that may not be there.
func usageOf(m *streamMessage) Usage {
	if m == nil {
		return Usage{}
	}

	return m.Usage.usage()
}

// addUsage is two counts, added field by field.
func addUsage(a, b Usage) Usage {
	return Usage{
		Input:      a.Input + b.Input,
		Output:     a.Output + b.Output,
		CacheRead:  a.CacheRead + b.CacheRead,
		CacheWrite: a.CacheWrite + b.CacheWrite,
	}
}
