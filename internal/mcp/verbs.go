package mcp

// The tools this server gets for free.
//
// internal/verb declares every action Orbit can be asked for. The tools
// written by hand in tools.go stay written by hand — their descriptions are
// paragraphs addressed to a model, and the names are what clients have
// registered — and everything in the declaration they do not already carry
// gets a tool built from it: the schema from what the verb takes, the
// sentence from what it says about itself, and the doing from the one body
// in internal/verb.
//
// spelledAs below is the join between the two lists. It is not an alias
// mechanism: nothing answers to two names. It is this package writing down
// which verb each of its own tools is, so that adding a verb to the
// declaration cannot quietly leave the model unable to ask for it.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// spelledAs is the verb each hand-written tool is.
//
// The names on the right are what the tools have always been called, and a
// model that has learned them should not have to learn them again for a
// rename that changes nothing about what they do.
func spelledAs() map[string]string {
	return map[string]string{
		"new":       "orbit_create_task",
		"run":       "orbit_retry_task",
		"list":      "orbit_list_tasks",
		"show":      "orbit_inspect_task",
		"note":      "orbit_add_note",
		"say":       "orbit_supervisor_say",
		"thread":    "orbit_supervisor_history",
		"knowledge": "orbit_knowledge",
		"learn":     "orbit_learn",
		"pause":     "orbit_pause_task",
		"cancel":    "orbit_cancel_task",
		"requeue":   "orbit_requeue_task",
		"direct":    "orbit_direct_task",
		"flows":     "orbit_list_flows",
		"flow":      "orbit_get_flow",
		"repos":     "orbit_list_repos",
	}
}

// built is every verb this server makes a tool of its own for: the whole
// declaration, less the ones a hand-written tool already carries and the
// ones it will not offer.
func built() []verb.Verb {
	hand := spelledAs()

	var out []verb.Verb

	for _, v := range verb.Every() {
		if hand[v.Path()] == "" && cannot[v.Path()] == "" {
			out = append(out, v)
		}
	}

	return out
}

// verbTools is a tool for every one of them.
func verbTools() []Tool {
	p := words.For("")

	out := make([]Tool, 0, len(built()))
	for _, v := range built() {
		out = append(out, toolFor(v, p))
	}

	return out
}

// cannot is a verb this server does not offer, and why.
//
// Four are a person's decision and not a model's, and one is a terminal —
// which is not something a tool call has to hand to anybody. The same list
// is in internal/arch's notThere, where it is argued with rather than just
// obeyed.
var cannot = map[string]string{
	"pr":       "opening a pull request is a person's decision",
	"pr merge": "merging is a person's decision",
	"pr close": "closing a pull request is a person's decision",
	"approve":  "accepting a library a task reached for is the question the gate asked a person",
	"take":     "this hands a terminal to an engine, and a tool call has no terminal to hand over",
}

// toolFor is one verb as a tool.
func toolFor(v verb.Verb, p *words.Printer) Tool {
	props := map[string]Property{}

	var need []string

	if v.OnTask {
		props["task_id"] = Property{Type: "string", Description: "The task's id, as orbit_list_tasks reports it."}

		need = append(need, "task_id")
	}

	for _, f := range v.Takes {
		props[f.Name] = Property{Type: kindOf(f.Kind), Description: f.About(p) + "."}
		if f.Needed {
			need = append(need, f.Name)
		}
	}

	return Tool{
		Name:        toolName(v),
		Description: strings.ToUpper(v.About(p)[:1]) + v.About(p)[1:] + ".",
		InputSchema: object(props, need...),
	}
}

// toolName is what a verb is called here: the prefix every tool of this
// server carries, and the whole of what the verb is called with the two
// characters a tool name may not hold spelled as underscores. A child is
// both of its words — orbit_rules_keep — because orbit_keep says nothing
// about what.
func toolName(v verb.Verb) string {
	return "orbit_" + strings.NewReplacer("-", "_", " ", "_").Replace(v.Path())
}

// kindOf is the JSON type a field arrives as.
func kindOf(k verb.Kind) string {
	if k == verb.YesOrNo {
		return "boolean"
	}

	return "string"
}

// askVerb runs one declared verb for a tool call.
func (sn Session) askVerb(name string, args map[string]any) CallToolResult {
	sb, err := sn.readBoard()
	if err != nil {
		return refuse(err)
	}

	defer sb.close()

	in := verb.In{Args: wordsOf(args), By: journalBy}

	if id := stringArg(args, "task_id"); id != "" {
		row, err := findTask(sb.board, id)
		if err != nil {
			return refuse(err)
		}

		in.Task, in.Repo = row.ID, row.RepoPath
	}

	out, err := verb.Run(sn.context(), sn.world(sb), name, in)
	if err != nil {
		return refuse(err)
	}

	if out.Saw != nil {
		return reply(out.Saw)
	}

	return done("%s", out.Said)
}

// wordsOf is what was sent, as the strings a verb takes.
//
// A client sends JSON, so a switch arrives as true and a count as a number;
// what a verb takes is text it parses itself, one way, for all four ways in.
func wordsOf(args map[string]any) map[string]string {
	out := make(map[string]string, len(args))

	for k, v := range args {
		switch t := v.(type) {
		case string:
			out[k] = t
		case bool, float64:
			out[k] = fmt.Sprint(t)
		}
	}

	return out
}
