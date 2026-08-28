package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/e1i0r/orbit/internal/flow"
)

// reply carries a tool's answer back as the one text block the protocol
// defines, holding indented JSON.
//
// JSON rather than prose, because the caller is a model that will act on the
// fields and then quote them: a sentence has to be parsed back out, and a
// model that parses a sentence wrong invents a task id. Indented rather than
// compact for the same reason a diff is: it is read by something that was
// trained on formatted JSON.
func reply(v any) CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		// Nothing this package builds can fail to encode — every value is a
		// map of strings, numbers and bools — so this is unreachable rather
		// than unhandled, and it says so instead of dropping the error.
		return refuse(fmt.Errorf("encode the answer: %w", err))
	}
	return CallToolResult{Content: []ContentItem{{Type: "text", Text: string(data)}}}
}

// refuse is a tool that ran and answered no.
//
// It is a result with isError rather than a JSON-RPC error object, because
// the model is the one that has to fix it: a client turns an error object
// into a transport failure the model never sees, and "there is no task
// ORB-9" is exactly the kind of thing the model should read and correct.
func refuse(err error) CallToolResult {
	return CallToolResult{
		Content: []ContentItem{{Type: "text", Text: err.Error()}},
		IsError: true,
	}
}

// done is a tool that acted and has one sentence to say about it.
func done(format string, args ...any) CallToolResult {
	return CallToolResult{Content: []ContentItem{{Type: "text", Text: fmt.Sprintf(format, args...)}}}
}

// stringArg reads one string argument, treating a missing one and an empty
// one as the same thing: a client that omits an optional field and one that
// sends "" mean the same, and telling them apart would make the tools
// behave differently for two clients that said nothing.
func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	s, ok := args[key].(string)
	if !ok {
		return ""
	}
	return s
}

// phaseNames is a flow's phases in order, with the engine each one runs on,
// which is what a supervisor deciding between flows actually wants.
func phaseNames(f flow.Flow) []map[string]any {
	phases := make([]map[string]any, 0, len(f.Phases))
	for _, p := range f.Phases {
		phases = append(phases, map[string]any{
			"name":   p.Name,
			"engine": p.Engine,
			"model":  p.Model,
			"waits":  p.Wait,
		})
	}
	return phases
}

// boolArg reads one boolean argument.
//
// Anything that is not a JSON true is false, which is the reading that fails
// safe: every boolean this server takes is a caller insisting on something
// destructive, and "delete_tasks": "yes" — a string, from a model that has
// seen both spellings — must not be the thing that deletes a record.
func boolArg(args map[string]any, key string) bool {
	yes, ok := args[key].(bool)
	return ok && yes
}
