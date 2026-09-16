package main

// What the engine writes on its way past, in the shape claude writes it.
//
// The lines here are the ones internal/engine reads: an init carrying the
// session id, an assistant message per thought, one per file written, and
// the result object that carries the answer, the cost and the count. A field
// that engine does not read is not written, and no field it does read is
// invented — internal/engine/testdata holds the real thing to compare
// against.

import (
	"encoding/json"
	"fmt"
	"os"
)

// say writes one turn to stdout as a stream of JSON lines.
func say(t turn, phase string) error {
	session := "fake-" + phase

	lines := []any{
		map[string]any{"type": "system", "subtype": "init", "session_id": session},
	}

	for i, thought := range t.Thoughts {
		lines = append(lines, message(fmt.Sprintf("msg_think_%d", i), map[string]any{
			"type": "thinking", "thinking": thought,
		}))
	}

	i := 0

	for path := range t.Write {
		lines = append(lines, message(fmt.Sprintf("msg_tool_%d", i), map[string]any{
			"type": "tool_use", "id": fmt.Sprintf("call_%d", i), "name": "Write",
			"input": map[string]any{"file_path": path},
		}))
		i++
	}

	lines = append(lines, map[string]any{
		"type":           "result",
		"subtype":        "success",
		"result":         t.Say,
		"session_id":     session,
		"total_cost_usd": t.Cost,
		"usage": map[string]any{
			"input_tokens": 1200, "output_tokens": 340,
			"cache_read_input_tokens": 18000, "cache_creation_input_tokens": 900,
		},
	})

	for _, line := range lines {
		body, err := json.Marshal(line)
		if err != nil {
			return fmt.Errorf("encode a line of the stream: %w", err)
		}

		if _, err := fmt.Fprintln(os.Stdout, string(body)); err != nil {
			return fmt.Errorf("write a line of the stream: %w", err)
		}
	}

	return nil
}

// message is one assistant record holding one content block, which is how
// claude sends a thought and how it sends a tool call.
func message(id string, block map[string]any) map[string]any {
	return map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"id":      id,
			"content": []any{block},
			"usage":   map[string]any{"input_tokens": 10, "output_tokens": 2},
		},
	}
}

// refused is a turn a headless run was denied: the tool call, the error that
// came back, and a result that says so and exits zero.
//
// The shape is the one internal/engine reads — a tool_result block marked
// is_error whose text says permission denied — and the wording is the
// provider's own. It is what a real engine does when the posture refuses it
// something and there is nobody to ask.
func refused(t turn, phase string) error {
	session := "fake-" + phase
	call := "call_refused"

	lines := []any{
		map[string]any{"type": "system", "subtype": "init", "session_id": session},
		message("msg_refused", map[string]any{
			"type": "tool_use", "id": call, "name": t.Refuse,
			"input": map[string]any{"file_path": "CONTRIBUTING.md"},
		}),
		map[string]any{
			"type": "user",
			"message": map[string]any{
				"content": []any{map[string]any{
					"type": "tool_result", "tool_use_id": call, "is_error": true,
					"content": t.Refuse + ": permission denied",
				}},
			},
		},
		map[string]any{
			"type": "result", "subtype": "success", "session_id": session,
			"result": "I could not write it: a permission prompt was not approved. " +
				"Could you approve it and ask me again?",
		},
	}

	for _, line := range lines {
		body, err := json.Marshal(line)
		if err != nil {
			return fmt.Errorf("encode a line of the stream: %w", err)
		}

		if _, err := fmt.Fprintln(os.Stdout, string(body)); err != nil {
			return fmt.Errorf("write a line of the stream: %w", err)
		}
	}

	return nil
}
