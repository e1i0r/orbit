package view

// An engine's output as a reader wants it.
//
// A phase that ran to the end writes prose: the model's answer, already
// markdown. A phase that was killed writes whatever its engine had put on
// stdout by the time it died, and for the three engines orbit drives that is
// a stream of JSON frames — hook traffic, token counters, tool calls, the
// whole skills prompt. Seventy kilobytes of it, with two sentences of the
// model's own words somewhere inside.
//
// Those two sentences are what this file finds. It reads the frames rather
// than the record being written differently on purpose: the record keeps
// what the engine actually printed, so `[v] raw` stays an honest answer, and
// a log written before this file existed reads as well as one written after.

import (
	"encoding/json"
	"strings"
)

// frame is one line of a stream, in the union of the three shapes orbit's
// engines write: claude nests the model's turn under message.content, codex
// puts one completed item on the line, opencode one part. Reading the union
// rather than switching on the engine is what lets a log say which words
// were the model's without also being told who ran it.
type frame struct {
	Type    string `json:"type"`
	Result  string `json:"result"`
	Message *turn  `json:"message"`
	Item    *block `json:"item"`
	Part    *block `json:"part"`
}

// turn is one message from the model, which is a list of blocks.
type turn struct {
	Content []block `json:"content"`
}

// block is one piece of a turn: what kind it is, and — when it is the kind a
// person reads — what it says.
type block struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// unframe is the model's own words with the stream's framing taken off, or
// the text unchanged when it was never framed.
func unframe(text string) string {
	lines := strings.Split(text, "\n")
	if !framed(lines) {
		return text
	}

	var out []string

	for _, l := range lines {
		f, ok := readFrame(l)
		if !ok {
			continue
		}

		// A result frame is the engine's own account of the whole run, so
		// it stands for the turns that led to it rather than beside them.
		if f.Type == "result" && strings.TrimSpace(f.Result) != "" {
			return strings.TrimSpace(f.Result)
		}

		if w := f.words(); w != "" {
			out = append(out, w)
		}
	}

	return strings.Join(out, "\n\n")
}

// framed says the text is a stream and not prose: its first line is a JSON
// object that names its own kind. Prose that opens with a brace is still
// prose, and a tool call's arguments are an object that never says what kind
// of thing it is.
func framed(lines []string) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}

		f, ok := readFrame(l)

		return ok && f.Type != ""
	}

	return false
}

// readFrame reads one line as a frame. A line that is not an object, or is
// an object this build cannot read, is not one — a stream half-written when
// the process died ends in a line like that.
func readFrame(line string) (frame, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "{") {
		return frame{}, false
	}

	var f frame
	if err := json.Unmarshal([]byte(line), &f); err != nil {
		return frame{}, false
	}

	return f, true
}

// words is what a person reads on one frame, and nothing else. Thinking,
// tool calls and their results are entries of the record in their own right
// and the task view draws them where they belong; repeating them here would
// put the same work on the screen twice, in the pane that is meant to say
// what the model made of it.
func (f frame) words() string {
	switch f.Type {
	case "assistant":
		return f.text()
	case "item.completed":
		if f.Item != nil && f.Item.Type == "agent_message" {
			return strings.TrimSpace(f.Item.Text)
		}
	case "text":
		if f.Part != nil && f.Part.Type == "text" {
			return strings.TrimSpace(f.Part.Text)
		}
	}

	return ""
}

// text is the text blocks of one turn, run together with the blank line
// between them that says a paragraph ended.
func (f frame) text() string {
	if f.Message == nil {
		return ""
	}

	var out []string

	for _, c := range f.Message.Content {
		if c.Type != "text" {
			continue
		}

		if words := strings.TrimSpace(c.Text); words != "" {
			out = append(out, words)
		}
	}

	return strings.Join(out, "\n\n")
}
