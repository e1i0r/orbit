package mcp

import "strings"

// Tools is every tool Orbit exposes, and the whole of what a supervising
// model can do through this server.
//
// A schema here is a promise the handler keeps. That is not a style note: an
// argument declared and then ignored is worse than an argument that does not
// exist, because a model passes it, reads a success, and believes it took
// effect. An earlier version of this list declared engine and model on
// orbit_create_task; neither reaches task.Create, which takes the engine
// from the phase of the flow, so both are gone rather than accepted and
// dropped.
func Tools() []Tool {
	return append(taskTools(), workspaceTools()...)
}

// taskTools is everything a supervisor does to a task: read the board, write
// one down, run it again, say something about it, stop it.
func taskTools() []Tool {
	return []Tool{
		{
			Name:        "orbit_get_board_summary",
			Description: "The state of the Orbit cockpit: how many repositories, how many tasks in each band (todo, needs_you, running, done), how many finished tasks nobody has read, and what the runs have cost so far.",
			InputSchema: object(nil),
		},
		{
			Name:        "orbit_list_tasks",
			Description: "Every task on the board, newest state first, optionally narrowed to one band or one repository. Start here: the ids it returns are what every other tool takes.",
			InputSchema: object(map[string]Property{
				"band": {Type: "string", Description: "Only tasks in this band.", Enum: bandNames()},
				"repo": {Type: "string", Description: "Only tasks in this repository, by name or by path."},
			}),
		},
		{
			Name:        "orbit_inspect_task",
			Description: "Everything the cockpit's inspector shows about one task: the text it was written with, the phases it walked and how each ended, its gate checks, the engine's recent thinking, the last error with the tail of what the engine printed, every note left on it, and what has already been done to it from outside a run — the calls other supervisors made and the sessions somebody opened by hand.",
			InputSchema: object(map[string]Property{
				"task_id": {Type: "string", Description: "The task's id, as orbit_list_tasks reports it."},
				"repo":    {Type: "string", Description: "Which repository, when two of them hold a task under this id."},
			}, "task_id"),
		},
		{
			Name:        "orbit_create_task",
			Description: "Write a new task down on the board. It is not started: writing a task and spending money running it are two decisions, and orbit_retry_task takes the second.",
			InputSchema: object(map[string]Property{
				"title":  {Type: "string", Description: "One line saying what the task is."},
				"prompt": {Type: "string", Description: "The instructions the engine is given, below the title."},
				"repo":   {Type: "string", Description: "Which repository to write it against, by name or by path. Only needed when Orbit knows more than one."},
				"flow":   {Type: "string", Description: "Which flow it walks, from orbit_list_flows. Defaults to the flow `orbit set flow` chose."},
				"id":     {Type: "string", Description: "The id to file it under. Defaults to the repository's name and the next free number."},
			}, "title"),
		},
		{
			Name:        "orbit_retry_task",
			Description: "Run a task that is not currently running — one that has never started, or one that failed. A corrective prompt is written into the task's record as a supervisor note before the run begins, so the next attempt and the reason for it are in one history.",
			InputSchema: object(map[string]Property{
				"task_id":           {Type: "string", Description: "The task's id."},
				"repo":              {Type: "string", Description: "Which repository, when two of them hold a task under this id."},
				"corrective_prompt": {Type: "string", Description: "What to do differently this time. Recorded as a note on the task."},
				"flow":              {Type: "string", Description: "Walk this flow instead of the one the task was written against."},
			}, "task_id"),
		},
		{
			Name:        "orbit_add_note",
			Description: "Leave a note on a task. It is appended to the task's record and shown in the cockpit's notes tab, marked as coming from a supervisor rather than from the person at the keyboard.",
			InputSchema: object(map[string]Property{
				"task_id": {Type: "string", Description: "The task's id."},
				"text":    {Type: "string", Description: "What to write down."},
				"repo":    {Type: "string", Description: "Which repository, when two of them hold a task under this id."},
			}, "task_id", "text"),
		},
		{
			Name:        "orbit_pause_task",
			Description: "Ask a running task to stop at its next phase boundary. The word is left for the run to read, so it takes effect when the current phase ends and not immediately.",
			InputSchema: object(map[string]Property{
				"task_id": {Type: "string", Description: "The task's id."},
				"repo":    {Type: "string", Description: "Which repository, when two of them hold a task under this id."},
			}, "task_id"),
		},
		{
			Name:        "orbit_cancel_task",
			Description: "Stop a run where it stands. The cancellation is written into the task's record.",
			InputSchema: object(map[string]Property{
				"task_id": {Type: "string", Description: "The task's id."},
				"repo":    {Type: "string", Description: "Which repository, when two of them hold a task under this id."},
			}, "task_id"),
		},
	}
}

// object builds one tool's input schema. Properties is never nil in the
// encoded form: a client that reads `"properties": null` for a tool taking
// no arguments has to guess whether that means none or unknown, and an empty
// object says none.
func object(props map[string]Property, required ...string) InputSchema {
	if props == nil {
		props = map[string]Property{}
	}
	return InputSchema{Type: "object", Properties: props, Required: required}
}

// toolNames is every tool's name, for the refusal a misspelled call gets.
func toolNames() []string {
	tools := Tools()
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return names
}

// instructions is what a client shows the model about this server before it
// has called anything, and it is deliberately about order rather than about
// capability: the tool list already says what each tool does, and what a
// model gets wrong is inventing a task id instead of reading one.
var instructions = strings.Join([]string{
	"Orbit is a cockpit for supervising coding agents. Tasks live on a board, in four bands: " + strings.Join(bandNames(), ", ") + ".",
	"Read before acting: orbit_get_board_summary for the shape of the board, orbit_list_tasks for the ids, orbit_inspect_task for why one of them failed.",
	"Never invent a task id or a repository path. Both come from orbit_list_tasks.",
	"orbit_retry_task and orbit_cancel_task change what is running and cost money. Say what you are about to do before you do it.",
	"orbit_forget_repo and orbit_delete_flow remove things that cannot be got back: a repository's record is the only account of what its runs did. Both refuse until told again.",
}, " ")
