package mcp

// Reading, writing and removing the flows a task can walk.
//
// A flow of the reader's own has always been a file in one directory, and
// until now a text editor was the only thing that could put one there. That
// is fine for a person sitting in front of the machine and impossible for
// anything reaching Orbit through a tool call — so a supervisor could choose
// between flows it had, and could not make the one the work wanted.

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
)

// originNames says where a flow came from, in words a program can branch on.
//
// internal/flow deliberately holds no English — a flow is data, and the
// sentences a reader sees are written at the call site that draws them — so
// the classification travels as a value and is named here, the same way the
// command line and the window each name it for their own screen.
var originNames = map[flow.Origin]string{
	flow.OriginBuiltin: "builtin",
	flow.OriginUser:    "user",
	flow.OriginShadow:  "shadow",
}

func originName(o flow.Origin) string {
	if name, ok := originNames[o]; ok {
		return name
	}

	return "unknown"
}

func (sn Session) listFlows() CallToolResult {
	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

	defer func() { _ = s.Close() }() //nolint:errcheck // the answer is already made

	listed := flow.List(s)

	flows := make([]map[string]any, 0, len(listed))
	for _, entry := range listed {
		f, err := flow.Resolve(s, entry.Name)
		if err != nil {
			// A flow that will not resolve is still a flow the reader has,
			// and naming it without its phases says more than dropping it.
			flows = append(flows, map[string]any{"name": entry.Name, "origin": originName(entry.Origin), "error": err.Error()})
			continue
		}

		flows = append(flows, map[string]any{
			"name":        entry.Name,
			"origin":      originName(entry.Origin),
			"description": f.Description,
			"phases":      phaseNames(f),
		})
	}

	return reply(map[string]any{"flows": flows, "dir": s.FlowDir()})
}

// getFlow answers one flow as the document orbit_save_flow takes, which is
// what makes "copy this and change one phase" a thing a model can do without
// being told the field names twice.
func (sn Session) getFlow(args map[string]any) CallToolResult {
	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

	defer func() { _ = s.Close() }() //nolint:errcheck // the answer is already made

	name := strings.TrimSpace(stringArg(args, "name"))
	if name == "" {
		return refuse(fmt.Errorf("this tool needs name"))
	}

	f, err := flow.Resolve(s, name)
	if err != nil {
		return refuse(err)
	}

	doc, err := flowDoc(f)
	if err != nil {
		return refuse(err)
	}

	answer := map[string]any{"flow": doc, "origin": originName(originOf(s, name))}
	if path, err := flow.UserPath(s, name); err == nil {
		answer["path"] = path
	}

	return reply(answer)
}

// saveFlow writes a flow down, either whole or as a change to one that is
// already there.
func (sn Session) saveFlow(args map[string]any) CallToolResult {
	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

	defer func() { _ = s.Close() }() //nolint:errcheck // the answer is already made

	name := strings.TrimSpace(stringArg(args, "name"))
	if name == "" {
		return refuse(fmt.Errorf("this tool needs name"))
	}

	doc, err := flowDocument(s, name, args)
	if err != nil {
		return refuse(err)
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		return refuse(fmt.Errorf("encode the flow %q: %w", name, err))
	}
	// Through the flow package's own decoder rather than into a Flow
	// directly: it refuses fields nobody declared, so a phase written with
	// "engines" is an error the caller reads instead of a phase saved with
	// no engine at all.
	f, err := flow.Decode(raw, name)
	if err != nil {
		return refuse(err)
	}

	path, err := flow.Save(s, f)
	if err != nil {
		return refuse(err)
	}

	return reply(map[string]any{
		"name":     f.Name,
		"path":     path,
		"phases":   phaseNames(f),
		"shadows":  slices.Contains(flow.BuiltinNames(), f.Name),
		"used_by":  sn.flowUsers(f.Name),
		"saved":    true,
		"walks_at": "the next run of a task written against this flow",
	})
}

// flowDocument is the flow a save is asking for, as JSON-shaped values:
// either the phases the caller wrote, or the phases of a flow it named.
func flowDocument(s flowSource, name string, args map[string]any) (map[string]any, error) {
	doc := map[string]any{"name": name}
	if description := strings.TrimSpace(stringArg(args, "description")); description != "" {
		doc["description"] = description
	}

	// Left out entirely when nobody said, rather than written as the
	// default: a flow file that carries "attempts": 3 has decided
	// something, and a caller who did not mention attempts has not. The two
	// read the same until the default changes.
	if n := intArg(args, "attempts"); n != 0 {
		doc["attempts"] = n
	}

	if phases, ok := args["phases"]; ok && phases != nil {
		doc["phases"] = phases
		return doc, nil
	}

	from := strings.TrimSpace(stringArg(args, "from"))
	if from == "" {
		return nil, fmt.Errorf("a flow needs phases: either give phases, or name a flow to copy with from")
	}

	base, err := flow.Resolve(s, from)
	if err != nil {
		return nil, err
	}

	doc["phases"] = base.Phases
	if _, ok := doc["description"]; !ok && base.Description != "" {
		doc["description"] = base.Description
	}

	return doc, nil
}

// flowSource is flow.Source, named here so this file's helpers do not have
// to take the whole store to answer a question about one directory.
type flowSource = flow.Source

// deleteFlow removes a flow the reader wrote.
func (sn Session) deleteFlow(args map[string]any) CallToolResult {
	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

	defer func() { _ = s.Close() }() //nolint:errcheck // the answer is already made

	name := strings.TrimSpace(stringArg(args, "name"))
	if name == "" {
		return refuse(fmt.Errorf("this tool needs name"))
	}
	// A shipped flow underneath means nothing is stranded: the name goes on
	// resolving, to what Orbit ships. Without one, a task written against
	// this name has nothing left to walk.
	shipped := slices.Contains(flow.BuiltinNames(), name)

	users := sn.flowUsers(name)
	if !shipped && len(users) > 0 && !boolArg(args, "force") {
		return refuse(fmt.Errorf("%d tasks are written against %q (%s) and nothing would resolve that name once it is gone; pass force to remove it anyway, or write the tasks against another flow first", len(users), name, strings.Join(users, ", ")))
	}

	revealed, err := flow.Delete(s, name)
	if err != nil {
		return refuse(err)
	}

	return reply(map[string]any{
		"name":             name,
		"deleted":          true,
		"restored_builtin": revealed,
		"used_by":          users,
	})
}

// flowUsers is every task written against a flow, as "repo/id".
//
// It is advisory and answers nothing when the board cannot be read: this is
// a warning about what a deletion would strand, and a board that will not
// fold is not a reason to refuse to remove a file the reader wrote.
func (sn Session) flowUsers(name string) []string {
	sb, err := sn.readBoard()
	if err != nil {
		return nil
	}

	defer sb.close()

	var users []string

	for _, t := range sb.board.Tasks {
		if t.Flow == name {
			users = append(users, t.Repo+"/"+t.ID)
		}
	}

	return users
}

// originOf is where the flow a name resolves to came from.
func originOf(s flowSource, name string) flow.Origin {
	for _, entry := range flow.List(s) {
		if entry.Name == name {
			return entry.Origin
		}
	}

	return flow.OriginUnknown
}

// flowDoc is a flow as the document that would be saved, built by encoding
// the flow itself rather than by listing its fields again here — a second
// list would be one more place for a field added to internal/flow to go
// missing without anything failing.
func flowDoc(f flow.Flow) (map[string]any, error) {
	raw, err := json.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("encode the flow %q: %w", f.Name, err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("read back the flow %q: %w", f.Name, err)
	}

	return doc, nil
}
