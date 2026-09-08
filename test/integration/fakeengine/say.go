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
