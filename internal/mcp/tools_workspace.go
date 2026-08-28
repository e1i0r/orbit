package mcp

// The tools that act on the workspace rather than on a task: the flows a
// task can walk, and the repositories Orbit knows.
//
// They are here and not in tools.go for the reason CONTRIBUTING gives, and
// they read as a group anyway: a supervisor that can write a flow and
// register a checkout can set up the work it is about to supervise, which is
// the difference between a model that files tasks and one that can be handed
// a machine.

// workspaceTools is the flow and repository half of the tool list.
func workspaceTools() []Tool {
	return append(flowTools(), repoTools()...)
}

func flowTools() []Tool {
	return []Tool{
		{
			Name:        "orbit_list_flows",
			Description: "Every flow a task can be written against, with the phases each one walks, the engine each phase runs on, and whether the flow is one Orbit ships or one the reader wrote.",
			InputSchema: object(nil),
		},
		{
			Name:        "orbit_get_flow",
			Description: "One flow in full: every phase with its engine, model, prompt, permissions and gate commands. This is the document orbit_save_flow takes, so reading one is how to copy it.",
			InputSchema: object(map[string]Property{
				"name": {Type: "string", Description: "The flow's name, as orbit_list_flows reports it."},
			}, "name"),
		},
		{
			Name:        "orbit_save_flow",
			Description: "Write a flow of the reader's own, or replace one. A flow saved under the name of one Orbit ships shadows it: tasks written against that name walk this one instead, and deleting it puts the shipped flow back. Copy an existing flow with `from` and change what needs changing rather than writing every phase out.",
			InputSchema: object(map[string]Property{
				"name":        {Type: "string", Description: "What the flow is called. It is also the file it is saved as."},
				"description": {Type: "string", Description: "One line saying what this flow is for."},
				"from":        {Type: "string", Description: "Start from this existing flow's phases. Ignored when phases are given."},
				"phases":      phasesProperty(),
			}, "name"),
		},
		{
			Name:        "orbit_delete_flow",
			Description: "Remove a flow the reader wrote. A flow Orbit ships cannot be deleted; deleting one that shadows a shipped flow restores the shipped one, which means tasks written against that name go on running, differently.",
			InputSchema: object(map[string]Property{
				"name":  {Type: "string", Description: "The flow to remove."},
				"force": {Type: "boolean", Description: "Remove it even though tasks are written against it and nothing would be left to resolve that name."},
			}, "name"),
		},
	}
}

// phasesProperty is the shape of a flow's phase list, spelled out because
// this is the one argument in the server a model has to compose rather than
// copy, and every field name here is one it would otherwise invent.
func phasesProperty() Property {
	return Property{
		Type:        "array",
		Description: "The steps the flow walks, in order. Each is run by one engine, and the next one starts when the last has finished and its gates have passed.",
		Items: &Property{
			Type:        "object",
			Description: "One step of the flow.",
			Required:    []string{"name", "engine"},
			Properties: map[string]Property{
				"name":        {Type: "string", Description: "What this step is called, as the record and the cockpit show it."},
				"engine":      {Type: "string", Description: "Which coding agent runs it."},
				"model":       {Type: "string", Description: "Which model that engine runs on. Its own default when unset."},
				"effort":      {Type: "string", Description: "How hard the model is asked to work, for engines that take that."},
				"thinking":    {Type: "string", Description: "How much thinking the model is asked for, for engines that take that."},
				"prompt":      {Type: "string", Description: "What this step is told to do, above the task's own text."},
				"feed_output": {Type: "boolean", Description: "Give this step what the previous one printed."},
				"wait":        {Type: "boolean", Description: "Stop for a human when this step ends, unless autopilot is on."},
				"permissions": {
					Type:        "array",
					Description: "What this step may touch. An empty list is the step that asks for nothing, and gets the most restrictive posture the engine can state.",
					Items:       &Property{Type: "string", Description: "One permission.", Enum: []string{"read", "repo", "network"}},
				},
				"gates": {
					Type:        "array",
					Description: "Commands run in the worktree after the engine finishes. A gate that exits non-zero fails the phase.",
					Items: &Property{
						Type:        "object",
						Description: "One check.",
						Required:    []string{"name", "command"},
						Properties: map[string]Property{
							"name":    {Type: "string", Description: "What the check is called."},
							"command": {Type: "string", Description: "The shell command to run."},
						},
					},
				},
			},
		},
	}
}

func repoTools() []Tool {
	return []Tool{
		{
			Name:        "orbit_list_repos",
			Description: "Every repository Orbit knows about, with the path each one is at, and the directories this server looked in to find them.",
			InputSchema: object(nil),
		},
		{
			Name:        "orbit_inspect_repo",
			Description: "One repository: where it is, which branch tasks are cut from, which remote it pushes to, and how its tasks stand — how many in each band, how many finished ones nobody has read, and what they have cost.",
			InputSchema: object(map[string]Property{
				"repo": {Type: "string", Description: "The repository, by name or by path."},
			}, "repo"),
		},
		{
			Name:        "orbit_add_repo",
			Description: "Tell Orbit about a repository, so that it is listed and searched without anything having been run in it. A path inside a checkout registers the checkout itself.",
			InputSchema: object(map[string]Property{
				"path": {Type: "string", Description: "Where the repository is on disk."},
			}, "path"),
		},
		{
			Name:        "orbit_forget_repo",
			Description: "Remove a repository's record from Orbit. This deletes the history of every task in it — the append-only record is the only account of what those runs did — so it refuses while any task is there unless delete_tasks says otherwise. Worktrees are left alone: they are checkouts git has registered, and `orbit cancel` is what removes one.",
			InputSchema: object(map[string]Property{
				"repo":         {Type: "string", Description: "The repository to forget, by name or by path."},
				"delete_tasks": {Type: "boolean", Description: "Delete the record of the tasks in it as well. Without this, a repository holding tasks is refused."},
			}, "repo"),
		},
	}
}
