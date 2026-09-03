package view

// The path the agent walked, pruned to the work.

import (
	"encoding/json"
	"strings"
)

// Step is one file the task changed: where it is, how many times it was
// changed, and how many times it was read on the way.
//
// The reads are kept as a count and not as steps of their own. A file opened
// four times before it was edited says something about how hard it was to
// find; four rows saying so would bury the file it is about.
type Step struct {
	Path    string
	Touches int
	Read    int
}

// edits are the tools that change a file. Anything else a model calls —
// reading, listing, running a build — is how it got there, not what it did.
var edits = map[string]bool{
	"Edit":         true,
	"MultiEdit":    true,
	"Write":        true,
	"NotebookEdit": true,
}

// Walk is the files a task changed, in the order it first reached them.
//
// This is the pruning rule the task story spec refuses to be built without,
// and it is the blunt form of it: out go the files that were opened and left
// alone, in stay all of the ones that changed — a hundred of them if there
// were a hundred. What is pruned is the noise; the work is never pruned.
//
// It is a reading of the record and not of a call graph. The agent walked
// the graph itself and every step it took is a phase.tool_call, so this
// works in any language and costs nothing — and it is weaker than a call
// graph, which is the trade the spec makes on purpose: it records what the
// agent looked at, not what calls what.
//
// The order is first touch, which is the order the agent got there. Sorting
// these by name would throw away the one thing the record knows that a diff
// does not.
func Walk(entries []Entry) []Step {
	var (
		out  []Step
		seen = map[string]int{}
	)

	for _, e := range entries {
		if e.What() != EntryToolCall {
			continue
		}

		path := filePath(e.Text)
		if path == "" {
			continue
		}

		at, known := seen[path]
		if !known {
			seen[path] = len(out)
			at = len(out)
			out = append(out, Step{Path: path})
		}

		if edits[e.Tool] {
			out[at].Touches++
			continue
		}

		out[at].Read++
	}

	return changed(out)
}

// changed drops the files nothing was done to, keeping the order of the rest.
func changed(steps []Step) []Step {
	kept := make([]Step, 0, len(steps))

	for _, s := range steps {
		if s.Touches > 0 {
			kept = append(kept, s)
		}
	}

	if len(kept) == 0 {
		return nil
	}

	return kept
}

// filePath is the file a tool call names, out of the arguments the engine
// wrote as JSON.
//
// Three spellings because three engines write it three ways, and one of them
// changing its mind is a walk that quietly goes empty rather than one that
// breaks. Arguments that are not JSON at all, or name no file — a build
// command, a search — are not files and say so by answering nothing.
func filePath(args string) string {
	var fields struct {
		FilePath string `json:"file_path"`
		Path     string `json:"path"`
		File     string `json:"file"`
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(args)), &fields); err != nil {
		return ""
	}

	for _, name := range []string{fields.FilePath, fields.Path, fields.File} {
		if name != "" {
			return name
		}
	}

	return ""
}
