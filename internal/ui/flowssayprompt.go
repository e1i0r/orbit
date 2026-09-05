package ui

// What the engine is told when somebody asks for a flow in words.

import "strings"

// flowDraftPrompt is that instruction: the shape of the document, the engines
// this build actually has, and the sentence to turn into one.
//
// The engine names are listed rather than left to the model, because a phase
// naming an engine this machine does not have is a flow that cannot run and
// a reader who has to work out why. The rules below are internal/flow's own,
// spelled out here because a draft that breaks one of them comes back as a
// decoder error rather than as a flow.
func flowDraftPrompt(said string, engines []string) string {
	var b strings.Builder

	b.WriteString("Write one Orbit flow as JSON, and answer with the JSON alone.\n\n")
	b.WriteString("A flow is a list of phases run in order. Each phase is one coding agent given instructions.\n\n")
	b.WriteString("The document:\n")
	b.WriteString(`{"name":"short-kebab-name","description":"one line","attempts":2,"phases":[` + "\n")
	b.WriteString(`  {"name":"1-implement","engine":"<engine>","model":"","thinking":"adaptive",` +
		`"permissions":["repo"],"feed_output":false,"wait":false,"prompt":"what this phase is told to do"},` + "\n")
	b.WriteString(`  {"name":"2-until-it-passes","loop":{"max":3,` +
		`"until":[{"name":"tests","command":"go test ./..."}],` +
		`"phases":[{"name":"fix","engine":"<engine>","feed_output":true,"permissions":["repo"],"prompt":"..."}]}}` + "\n")
	b.WriteString("]}\n\n")

	b.WriteString("The rules:\n")
	b.WriteString("- engine is one of: " + strings.Join(engines, ", ") + ". Use the same one throughout unless told otherwise.\n")
	b.WriteString("- a phase has an engine or a loop, never both; the loop's own phases have the engine.\n")
	b.WriteString("- a loop needs max and at least one check in until; a check is a shell command that exits non-zero when the work is not done.\n")
	b.WriteString("- never let a model decide when a loop is finished: only a command's exit code says so.\n")
	b.WriteString("- wait: true stops the flow for a human, and belongs on a review phase if there is one.\n")
	b.WriteString("- permissions are read, repo or network, and ask for the least that will do.\n")
	b.WriteString("- prompts are instructions to a coding agent, in the language the request below is written in.\n")
	b.WriteString("- leave model and effort empty unless the request names one.\n")
	b.WriteString("- it must be valid JSON: no comments, no trailing commas, and a line break inside a string is \\n and never a real one.\n\n")

	b.WriteString("The request:\n")
	b.WriteString(said)
	b.WriteString("\n")

	return b.String()
}
