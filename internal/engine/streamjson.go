package engine

// The shapes claude writes on its stream, spelled as claude spells them.

import (
	"encoding/json"
	"strings"
)

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
	// ID is the message the count belongs to. claude repeats one assistant
	// message across several records with the identical usage object, so
	// the running sum counted the same turn as many times as it was
	// repeated.
	ID      string          `json:"id"`
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
	ID       string          `json:"id"`
	Input    json.RawMessage `json:"input"`
	// ToolUseID is what a tool_result names the call it answers. The result
	// carries no tool name of its own — claude sets none on 851 blocks out
	// of 851 — so this is the only way back to what was refused.
	ToolUseID string `json:"tool_use_id"`
	// Content is a string on some blocks and an array of blocks on others.
	// Typed as a string, the whole envelope failed to unmarshal and the line
	// was dropped, which lost 75 of those 851 blocks and the refusals among
	// them.
	Content json.RawMessage `json:"content"`
	IsError bool            `json:"is_error"`
}

// said is a tool_result's content as text, whichever of its two shapes it
// arrived in.
func (c streamContent) said() string {
	if len(c.Content) == 0 {
		return ""
	}

	var one string
	if err := json.Unmarshal(c.Content, &one); err == nil {
		return one
	}

	var blocks []streamContent
	if err := json.Unmarshal(c.Content, &blocks); err != nil {
		return ""
	}

	var parts []string

	for _, b := range blocks {
		if b.Text != "" {
			parts = append(parts, b.Text)
		}
	}

	return strings.Join(parts, "\n")
}

type streamDelta struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Thinking string `json:"thinking"`
}
