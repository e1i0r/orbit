package flow

// Reading a flow out of whatever a model printed.
//
// A model asked for JSON wraps it in prose, in fences, in an apology, and
// sometimes prints a second object to explain the first. This is the one
// place that knows how to get the document back out of that, so the window
// and the tools that ask for a draft do not each keep their own guess.

// Taking everything between the first brace and the last swallowed the prose
// with it and handed the decoder a document with a paragraph in the middle —
// which is how a reader ended up being told about a field called
// "\n\ndescription" they never typed. So the braces are matched instead, and
// every balanced object in the answer is a candidate: the one with phases in
// it is the flow, and the rest is whatever the engine felt like saying.

import (
	"encoding/json"
	"fmt"
	"strings"
)

// jsonObjects is every balanced {...} in the answer, outermost first and in
// the order they were printed.
//
// Quotes are respected, so a brace inside a prompt does not open a document
// and a quotation mark inside prose does not hide one.
func jsonObjects(out string) []string {
	var (
		found    []string
		depth    int
		start    int
		inString bool
		escaped  bool
	)

	for i, r := range out {
		switch {
		case escaped:
			escaped = false
		case inString && r == '\\':
			escaped = true
		case r == '"':
			inString = !inString
		case inString:
		case r == '{':
			if depth == 0 {
				start = i
			}

			depth++
		case r == '}':
			if depth > 0 {
				depth--
				if depth == 0 {
					found = append(found, out[start:i+1])
				}
			}
		}
	}

	return found
}

// flowJSON is the object among them that looks like a flow: the first one
// with phases in it, and failing that the longest, which is the best guess
// left when the answer is broken enough that nothing parses.
func flowJSON(out string) (string, bool) {
	objects := jsonObjects(out)

	longest := ""

	for _, raw := range objects {
		var doc map[string]any
		if json.Unmarshal([]byte(raw), &doc) == nil {
			if _, held := doc["phases"]; held {
				return raw, true
			}
		}

		if len(raw) > len(longest) {
			longest = raw
		}
	}

	if longest == "" {
		return "", false
	}

	return longest, true
}

// fenced strips a markdown fence when the whole answer is inside one, which
// is what an engine does on the days it ignores being asked not to.
func fenced(out string) string {
	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "```") {
		return out
	}

	_, rest, held := strings.Cut(trimmed, "\n")
	if !held {
		return out
	}

	if end := strings.LastIndex(rest, "```"); end >= 0 {
		return rest[:end]
	}

	return rest
}

// mendJSON: reading JSON an engine wrote by hand.
//
// A model asked for JSON writes a prompt with line breaks in it and leaves
// them raw inside the string, which is not JSON — the decoder says "invalid
// character '\n' in string literal" and the reader is left with a refusal
// and a wall of text they cannot fix. Since what is wrong is exactly one
// thing, and mending it changes no valid document, it is mended here rather
// than handed back.

// mendJSON escapes the control characters an engine left raw inside a string.
//
// Outside a string every one of them is whitespace and stays as it is. A
// well-formed document cannot hold them inside one, so this is a no-op on
// anything a machine wrote.
func mendJSON(raw []byte) []byte {
	out := make([]byte, 0, len(raw)+16)

	var (
		inString bool
		escaped  bool
	)

	for _, b := range raw {
		switch {
		case escaped:
			escaped = false
		case b == '\\' && inString:
			escaped = true
		case b == '"':
			inString = !inString
		case inString:
			if mended, ok := escapeIn(b); ok {
				out = append(out, mended...)
				continue
			}
		}

		out = append(out, b)
	}

	return out
}

// escapeIn is the two-character form of a control character, for the ones
// that turn up in a prompt somebody dictated.
func escapeIn(b byte) (string, bool) {
	switch b {
	case '\n':
		return `\n`, true
	case '\r':
		return `\r`, true
	case '\t':
		return `\t`, true
	}

	return "", false
}

// Draft is the flow an engine's answer holds.
func Draft(out string) (Flow, error) {
	found, held := flowJSON(fenced(out))
	if !held {
		return Flow{}, fmt.Errorf("the engine answered with no flow in it: %s", opening(out))
	}

	raw := asJSON([]byte(found))

	// A draft with no name of its own is given one rather than refused: the
	// name is what the reader types next anyway, and "the flow has no name"
	// tells them nothing about the flow they just asked for.
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Flow{}, fmt.Errorf("the engine's answer is not a flow: %w", err)
	}

	name, named := doc["name"].(string)
	if !named || strings.TrimSpace(name) == "" {
		name = "draft"
		doc["name"] = name

		filled, err := json.Marshal(doc)
		if err != nil {
			return Flow{}, fmt.Errorf("read the engine's answer back: %w", err)
		}

		raw = filled
	}

	return decode(raw, name)
}

// asJSON is the answer as written when that parses, and mended when it does
// not.
//
// Mending first would be a repair applied to documents that never needed one,
// and this one — escaping raw control characters inside strings — cannot tell
// a string that ran on from a string that was never closed. So it is the
// fallback and not the first move.
func asJSON(raw []byte) []byte {
	var any map[string]any
	if json.Unmarshal(raw, &any) == nil {
		return raw
	}

	return mendJSON(raw)
}

// opening is enough of an answer to say what came back instead of a flow,
// without printing a page of it into a one-line message.
func opening(out string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	if len(line) > 120 {
		return line[:120] + "…"
	}

	return line
}
